FROM alpine

ADD *.go /src/

RUN apk --update --no-cache add ca-certificates git go \
  && export GOPATH=/go \
  && REPO_PATH="github.com/george-angel/k8s-oidc-helper" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv src/* $GOPATH/src/${REPO_PATH} \
  && rm -rf src \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get ./... \
  && go build \
  && mv k8s-oidc-helper /k8s-oidc-helper \
  && apk del go git \
  && rm -rf $GOPATH /var/cache/apk/*

CMD [ "/k8s-oidc-helper" ]
