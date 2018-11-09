package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/cbroglie/mustache"
	"github.com/openshift/library-go/pkg/git"
	s2igit "github.com/openshift/source-to-image/pkg/scm/git"
	"github.com/spf13/cobra"
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

func createCommand() *cobra.Command {
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
			template, err := ioutil.ReadFile("/spec.mustache")
			if err != nil {
				return err
			}
			m := map[string]string{
				"image":       opt.image,
				"imageStream": opt.imageStream,
				"name":        opt.name,
				"revision":    revision,
				"source":      source,
				"toDocker":    fmt.Sprintf("%v", opt.toDocker),
			}
			yaml, err := mustache.Render(string(template), m)
			if err != nil {
				return err
			}
			fmt.Println("Applying buildConfig:", yaml)
			c := exec.Command("oc", "apply", "-f", "-")
			c.Stdin = strings.NewReader(yaml)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
	cmd.Flags().StringVar(&opt.name, "name", "", "build configuration name")
	cmd.Flags().StringVar(&opt.image, "image", "", "image name to push")
	cmd.Flags().StringVar(&opt.imageStream, "image-stream", "", "image stream to use as build input")
	cmd.Flags().BoolVar(&opt.toDocker, "to-docker", true, "push the image to a docker registry")
	return cmd
}
