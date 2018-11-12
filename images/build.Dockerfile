FROM docker.io/vdemeester/kobw-base:latest
COPY . /src/github.com/vdemeester/kotw
ENV GOPATH=/
RUN go build -o /usr/bin/kotw github.com/vdemeester/kotw
ENTRYPOINT ["/usr/bin/kotw"]
