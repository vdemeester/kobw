FROM centos:7
RUN yum -y install git golang && yum clean all
COPY . /src/github.com/vdemeester/kotw
ENV GOPATH=/
RUN go build -o /usr/bin/kotw github.com/vdemeester/kotw
COPY ./bin/oc /usr/bin/oc
COPY ./spec.mustache /spec.mustache
ENTRYPOINT ["/usr/bin/build"]
