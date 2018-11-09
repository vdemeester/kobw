bin:
	mkdir -p bin

bin/oc: bin
	./download_oc.sh

bin/kobw: bin
	go build -o bin/kotw -v .

.PHONY: vdemeester/oc-builder
vdemeester/oc-builder: bin/oc
	tar cf - bin/oc *.go vendor *.mustache build.sh Dockerfile | docker build -t vdemeester/oc-builder -
