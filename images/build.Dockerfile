FROM docker.io/vdemeester/kobw-base:latest
COPY . /src/github.com/vdemeester/kobw
ENV GOPATH=/
RUN go build -o /usr/bin/kobw github.com/vdemeester/kobw
ENTRYPOINT ["/usr/bin/kobw"]
