package main

import (
	"os"

	"github.com/docker/distribution/reference"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	buildclientsv1 "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	imageclientsv1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	corev1 "k8s.io/api/core/v1"
	kuberrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

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
