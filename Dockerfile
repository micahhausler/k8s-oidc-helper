FROM golang:alpine AS build
WORKDIR /go/src/github.com/utilitywarehouse/k8s-oidc-helper
COPY . /go/src/github.com/utilitywarehouse/k8s-oidc-helper
RUN apk --no-cache add git &&\
 go get ./... &&\
 CGO_ENABLED=0 go build -o /kubernetes-auth-conf .

FROM alpine:3.8
RUN apk add --no-cache ca-certificates
COPY --from=build /kubernetes-auth-conf /kubernetes-auth-conf
CMD [ "/kubernetes-auth-conf" ]
