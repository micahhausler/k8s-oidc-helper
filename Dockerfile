FROM golang:1.9

MAINTAINER Micah Hausler, <hausler.m@gmail.com>

WORKDIR /build
ADD . $WORKDIR

RUN go get github.com/ogier/pflag
RUN go get gopkg.in/yaml.v2
RUN CGO_ENABLED=0 go build -v -a -tags netgo -installsuffix netgo -ldflags '-w' -o /bin/k8s-oidc-helper .

FROM golang:1.9.0-alpine

WORKDIR /bin

COPY --from=0 /bin/k8s-oidc-helper .
RUN chmod 755 /bin/k8s-oidc-helper

ENTRYPOINT ["/bin/k8s-oidc-helper"]
