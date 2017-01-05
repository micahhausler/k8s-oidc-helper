FROM alpine:3.5

ENV GOPATH=/go

WORKDIR /go/src/app
ADD . /go/src/app/

RUN apk --update --no-cache add ca-certificates git go musl-dev \
  && go get ./... \
  && CGO_ENABLED=0 go build -ldflags '-s -extldflags "-static"' -o /kubernetes-auth-conf . \
  && apk del go git musl-dev \
  && rm -rf $GOPATH /var/cache/apk/*

CMD [ "/kubernetes-auth-conf" ]
