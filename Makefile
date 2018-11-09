default: bin/oc bin/kobw

bin:
	mkdir -p bin

bin/oc: bin
	./download_oc.sh

bin/kobw: bin
	go build -o bin/kotw -v .

.PHONY: vdemeester/oc-builder
vdemeester/oc-builder: bin/oc
	tar cf - bin/oc *.go vendor *.mustache build.sh Dockerfile | docker build -t vdemeester/oc-builder -

.PHONY: clean
clean:
	oc delete buildtemplate.build.knative.dev,build.build.knative.dev,build,buildconfig --all

.PHONY: run-build
run-build:
	ko apply -f config/

