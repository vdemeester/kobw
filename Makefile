default: bin/oc bin/kobw

bin:
	mkdir -p bin

bin/oc: bin
	./download_oc.sh bin

assets.go:
	go generate

bin/kobw: bin bin/oc assets.go
	go build -o bin/kotw -v .

.PHONY: docker.io/vdemeester/kobw-builder
docker.io/vdemeester/kobw-builder: docker.io/vdemeester/kobw-base
	tar cf - bin/oc images/build.Dockerfile *.go vendor | docker build -t $@ -f images/build.Dockerfile -

.PHONY: docker.io/vdemeester/kobw-base
docker.io/vdemeester/kobw-base: bin/oc
	tar cf - bin/oc images/base.Dockerfile | docker build -t $@ -f images/base.Dockerfile -

.PHONY: clean
clean:
	oc delete buildtemplate.build.knative.dev,build.build.knative.dev,build,buildconfig --all

.PHONY: run-build
run-build: bin/kobw
	ko apply -f config/

