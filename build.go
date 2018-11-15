package main

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/distribution/reference"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	buildclientsv1 "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	imageclientsv1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	buildclientmanual "github.com/openshift/origin/pkg/build/client/v1"
	ocerrors "github.com/openshift/origin/pkg/oc/lib/errors"
	corev1 "k8s.io/api/core/v1"
	kuberrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
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

func startBuild(config *rest.Config, name string) error {
	ns := getCurrentNamespace()
	buildV1Client, err := buildclientsv1.NewForConfig(config)
	if err != nil {
		return err
	}

	buildRequest := &buildv1.BuildRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		TriggeredBy: []buildv1.BuildTriggerCause{
			{Message: "Triggered by kobw"},
		},
	}
	b, err := buildV1Client.BuildConfigs(ns).Instantiate(name, buildRequest)
	if err != nil {
		return err
	}
	fmt.Println("name", b.Name)

	return waitForBuildComplete(buildV1Client.Builds(ns), b.Name)
	// return logAndWait(buildV1Client, ns, b.Name)
}

func logAndWait(buildV1Client *buildclientsv1.BuildV1Client, ns, buildName string) error {
	opts := buildv1.BuildLogOptions{
		Follow: true,
		NoWait: false,
	}
	buildLogClient := buildclientmanual.NewBuildLogClient(buildV1Client.RESTClient(), ns)
	var err error
	for {
		rd, logErr := buildLogClient.Logs(buildName, opts).Stream()
		if logErr != nil {
			err = ocerrors.NewError("unable to stream the build logs").WithCause(logErr)
			fmt.Println("log error", err)
			continue
		}
		defer rd.Close()
		if _, streamErr := io.Copy(os.Stderr, rd); streamErr != nil {
			err = ocerrors.NewError("unable to stream the build logs").WithCause(streamErr)
			fmt.Println("log error", err)
		}
		break
	}
	if err != nil {
		return err
	}
	return waitForBuildComplete(buildV1Client.Builds(ns), buildName)
}

func waitForBuildComplete(c buildclientsv1.BuildInterface, name string) error {
	isOK := func(b *buildv1.Build) bool {
		return b.Status.Phase == buildv1.BuildPhaseComplete
	}
	isFailed := func(b *buildv1.Build) bool {
		return b.Status.Phase == buildv1.BuildPhaseFailed ||
			b.Status.Phase == buildv1.BuildPhaseCancelled ||
			b.Status.Phase == buildv1.BuildPhaseError
	}
	for {
		list, err := c.List(metav1.ListOptions{FieldSelector: fields.Set{"metadata.name": name}.AsSelector().String()})
		if err != nil {
			return err
		}
		for i := range list.Items {
			if name == list.Items[i].Name && isOK(&list.Items[i]) {
				return nil
			}
			if name != list.Items[i].Name || isFailed(&list.Items[i]) {
				return fmt.Errorf("the build %s/%s status is %q", list.Items[i].Namespace, list.Items[i].Name, list.Items[i].Status.Phase)
			}
		}

		rv := list.ResourceVersion
		w, err := c.Watch(metav1.ListOptions{FieldSelector: fields.Set{"metadata.name": name}.AsSelector().String(), ResourceVersion: rv})
		if err != nil {
			return err
		}
		defer w.Stop()

		for {
			val, ok := <-w.ResultChan()
			if !ok {
				// reget and re-watch
				break
			}
			if e, ok := val.Object.(*buildv1.Build); ok {
				if name == e.Name && isOK(e) {
					return nil
				}
				if name != e.Name || isFailed(e) {
					return fmt.Errorf("The build %s/%s status is %q", e.Namespace, name, e.Status.Phase)
				}
			}
		}
	}
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
	obc, err := buildV1Client.BuildConfigs(ns).Get(opt.name, metav1.GetOptions{})
	if err != nil {
		if !kuberrors.IsNotFound(err) {
			return err
		}
		if _, err := buildV1Client.BuildConfigs(ns).Create(bc); err != nil {
			return err
		}
	} else {
		bc.ResourceVersion = obc.ResourceVersion
		if _, err := buildV1Client.BuildConfigs(ns).Update(bc); err != nil {
			return err
		}
	}
	return nil
}
