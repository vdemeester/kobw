package main

import (
	"fmt"
	"os"

	"github.com/docker/distribution/reference"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	buildclientsv1 "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	imageclientsv1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/openshift/library-go/pkg/git"
	s2igit "github.com/openshift/source-to-image/pkg/scm/git"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kuberrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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

func getCurrentNamespace() string {
	return os.Getenv("NAMESPACE") // FIXME(vdemeester) try more things to get the namespace (like /var/run/secrets/kubernetes.io/serviceaccount/namespace)
}

func createImageStreamIfNeeded(config *rest.Config, name string) error {
	ns := getCurrentNamespace()
	isClient, err := imageclientsv1.NewForConfig(config)
	if err != nil {
		return err
	}
	ref, err := reference.Parse(name)
	if err != nil {
		return err
	}
	image := ref.(reference.Named).Name() // FIXME(vdemeester) make this more robust
	_, err = isClient.ImageStreams(ns).Get(image, metav1.GetOptions{})
	if err != nil {
		if !kuberrors.IsNotFound(err) {
			return err
		}
		_, err = isClient.ImageStreams(ns).Create(&imagev1.ImageStream{
			ObjectMeta: metav1.ObjectMeta{
				Name:      image,
				Namespace: ns,
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func createBuildConfig(config *rest.Config, path string, opt createOption) error {
	source, revision, err := detectSource(path)
	if err != nil {
		return err
	}

	ns := getCurrentNamespace()
	buildV1Client, err := buildclientsv1.NewForConfig(config)
	if err != nil {
		return err
	}

	buildsource := buildv1.BuildSource{
		Type: buildv1.BuildSourceGit,
		Git: &buildv1.GitBuildSource{
			URI: source,
			Ref: revision,
		},
	}
	var buildoutput buildv1.BuildOutput
	if opt.toDocker {
		buildoutput = buildv1.BuildOutput{
			PushSecret: &corev1.LocalObjectReference{
				Name: "dockerhub",
			},
			To: &corev1.ObjectReference{
				Kind: "DockerImage",
				Name: opt.image,
			},
		}
	} else {
		buildoutput = buildv1.BuildOutput{
			To: &corev1.ObjectReference{
				Kind: "ImageStreamTag",
				Name: opt.image,
			},
		}
	}
	buildstrategy := buildv1.BuildStrategy{
		Type: buildv1.SourceBuildStrategyType,
		SourceStrategy: &buildv1.SourceBuildStrategy{
			From: corev1.ObjectReference{
				Kind:      "ImageStreamTag",
				Name:      opt.imageStream,
				Namespace: "openshift",
			},
		},
	}
	bc := &buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"openshift.io/generated-by": "kobw",
			},
			Labels: map[string]string{
				"name": opt.name,
			},
			Name: opt.name,
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Source:   buildsource,
				Output:   buildoutput,
				Strategy: buildstrategy,
			},
		},
	}
	_, err = buildV1Client.BuildConfigs(ns).Get(opt.name, metav1.GetOptions{})
	if err != nil {
		if !kuberrors.IsNotFound(err) {
			return err
		}
		if _, err := buildV1Client.BuildConfigs(ns).Create(bc); err != nil {
			return err
		}
	} else {
		if _, err := buildV1Client.BuildConfigs(ns).Update(bc); err != nil {
			return err
		}
	}
	return nil
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
