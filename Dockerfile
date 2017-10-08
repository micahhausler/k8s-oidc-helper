FROM golang:1.9

MAINTAINER Micah Hausler, <hausler.m@gmail.com>

WORKDIR /build
ADD . $WORKDIR

RUN go get github.com/ogier/pflag
RUN go get gopkg.in/yaml.v2
RUN go build -v -a -tags netgo -installsuffix netgo -ldflags '-w' -o /bin/k8s-oidc-helper .

FROM busybox

WORKDIR /bin

COPY --from=0 /bin/k8s-oidc-helper .
RUN chmod 755 /bin/k8s-oidc-helper

ENTRYPOINT ["/bin/k8s-oidc-helper"]
