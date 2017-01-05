FROM alpine:3.5

ADD *.go /src/

RUN apk --update --no-cache add ca-certificates git go musl-dev \
  && export GOPATH=/go \
  && REPO_PATH="github.com/utilitywarehouse/k8s-oidc-helper" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv src/* $GOPATH/src/${REPO_PATH} \
  && rm -rf src \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get ./... \
  && CGO_ENABLED=0 go build -ldflags '-s -extldflags "-static"' \
  && mv k8s-oidc-helper /k8s-oidc-helper \
  && apk del go git musl-dev \
  && rm -rf $GOPATH /var/cache/apk/*

CMD [ "/k8s-oidc-helper" ]
