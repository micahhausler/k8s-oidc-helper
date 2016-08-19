# To build:
# $ docker run --rm -v $(pwd):/go/src/github.com/micahhausler/k8s-oidc-helper -w /go/src/github.com/micahhausler/k8s-oidc-helper golang:1.7  go build -v -a -tags netgo -installsuffix netgo -ldflags '-w'
# $ docker build -t micahhausler/k8s-oidc-helper .
#
# To run:
# $ docker run micahhausler/k8s-oidc-helper

FROM busybox

MAINTAINER Micah Hausler, <hausler.m@gmail.com>

COPY k8s-oidc-helper /bin/k8s-oidc-helper
RUN chmod 755 /bin/k8s-oidc-helper

ENTRYPOINT ["/bin/k8s-oidc-helper"]
