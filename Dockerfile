FROM golang:1.9

MAINTAINER Micah Hausler, <hausler.m@gmail.com>

WORKDIR /build
ADD . $WORKDIR

RUN go get github.com/ogier/pflag
RUN go get gopkg.in/yaml.v2
RUN go build -v -a -tags netgo -installsuffix netgo -ldflags '-w' -o /bin/k8s-oidc-helper .

CMD /bin/k8s-oidc-helper
