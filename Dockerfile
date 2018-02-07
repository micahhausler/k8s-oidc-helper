FROM golang:1.9-alpine as builder

RUN apk add -U ca-certificates

ENV PKG=/go/src/github.com/micahhausler/k8s-oidc-helper
ADD . $PKG
WORKDIR $PKG

RUN go install -ldflags '-w'

FROM alpine:latest

RUN apk add -U ca-certificates

COPY --from=builder /go/bin/k8s-oidc-helper /bin/k8s-oidc-helper

ENTRYPOINT ["/bin/k8s-oidc-helper"]
