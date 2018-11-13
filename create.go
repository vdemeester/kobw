package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cbroglie/mustache"
	"github.com/docker/distribution/reference"
	v1 "github.com/openshift/api/image/v1"
	imagev1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/openshift/library-go/pkg/git"
	s2igit "github.com/openshift/source-to-image/pkg/scm/git"
	"github.com/spf13/cobra"
	kuberrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type createOption struct {
	name        string
	image       string
	imageStream string
	toDocker    bool
}

func validateCreateOpts(opt createOption) error {
	if opt.name == "" {
		return errors.New("name is empty")
	}
	if opt.image == "" {
		return errors.New("image is empty")
	}
	if opt.imageStream == "" {
		return errors.New("image-stream is empty")
	}
	return nil
}

func createCommand(opts kobwOptions) *cobra.Command {
	var opt createOption
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create or update the build config",
		RunE: func(cmd *cobra.Command, args []string) error {
			// debug()
			if err := validateCreateOpts(opt); err != nil {
				return err
			}
			s := args[0]
			var source string
			var revision string
			url, err := s2igit.Parse(s)
			if err != nil {
				return err
			}
			gitRepo := git.NewRepository()
			if url.IsLocal() {
				remote, ok, err := gitRepo.GetOriginURL(s)
				if err != nil && err != git.ErrGitNotAvailable {
					return err
				}
				if !ok {
					return fmt.Errorf("source is not supported %s (git: %s, %s)", s, url, remote)
				}
				source = remote
				revision = gitRepo.GetRef(s)
			} else {
				source = s
				revision = "master"
			}
			template, err := FSByte(false, "/assets/spec.mustache")
			if err != nil {
				return err
			}
			m := map[string]interface{}{
				"image":       opt.image,
				"imageStream": opt.imageStream,
				"name":        opt.name,
				"revision":    revision,
				"source":      source,
				"toDocker":    opt.toDocker,
			}
			yaml, err := mustache.Render(string(template), m)
			if err != nil {
				return err
			}
			config, err := clientcmd.BuildConfigFromFlags(opts.masterURL, opts.kubeconfig)
			if err != nil {
				return err
			}
			fmt.Println("Applying buildConfig:", yaml)
			c := exec.Command("oc", "apply", "-f", "-")
			c.Stdin = strings.NewReader(yaml)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				return err
			}
			if !opt.toDocker {
				ns := os.Getenv("NAMESPACE") // FIXME(vdemeester) try more things to get the namespace (like /var/run/secrets/kubernetes.io/serviceaccount/namespace)
				isClient, err := imagev1.NewForConfig(config)
				if err != nil {
					return err
				}
				ref, err := reference.Parse(opt.image)
				if err != nil {
					return err
				}
				image := ref.(reference.Named).Name() // FIXME(vdemeester) make this more robust
				_, err = isClient.ImageStreams(ns).Get(image, metav1.GetOptions{})
				if err != nil {
					if !kuberrors.IsNotFound(err) {
						return err
					}
					_, err = isClient.ImageStreams(ns).Create(&v1.ImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      image,
							Namespace: ns,
						},
					})
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opt.name, "name", "", "build configuration name")
	cmd.Flags().StringVar(&opt.image, "image", "", "image name to push")
	cmd.Flags().StringVar(&opt.imageStream, "image-stream", "", "image stream to use as build input")
	cmd.Flags().BoolVar(&opt.toDocker, "to-docker", true, "push the image to a docker registry")
	return cmd
}
