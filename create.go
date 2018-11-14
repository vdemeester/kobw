package main

import (
	"fmt"

	"github.com/openshift/library-go/pkg/git"
	s2igit "github.com/openshift/source-to-image/pkg/scm/git"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
			config, err := clientcmd.BuildConfigFromFlags(opts.masterURL, opts.kubeconfig)
			if err != nil {
				return errors.Wrap(err, "could not create kubernetes client config")
			}
			if err := createBuildConfig(config, args[0], opt); err != nil {
				return errors.Wrap(err, "failed to create BuildConfig")
			}
			if !opt.toDocker {
				if err := createImageStreamIfNeeded(config, opt.image); err != nil {
					return errors.Wrapf(err, "failed to create imagestream %s", opt.image)
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

func detectSource(path string) (string, string, error) {
	var source string
	var revision string
	url, err := s2igit.Parse(path)
	if err != nil {
		return "", "", err
	}
	gitRepo := git.NewRepository()
	if url.IsLocal() {
		remote, ok, err := gitRepo.GetOriginURL(path)
		if err != nil && err != git.ErrGitNotAvailable {
			return "", "", errors.Wrap(err, "could not detect source")
		}
		if !ok {
			return "", "", fmt.Errorf("source is not supported %s (git: %s, %s)", path, url, remote)
		}
		source = remote
		revision = gitRepo.GetRef(path)
	} else {
		source = path
		revision = "master"
	}
	return source, revision, nil
}
