[![Docker Build Status](https://img.shields.io/docker/build/micahhausler/k8s-oidc-helper.svg)](https://hub.docker.com/r/micahhausler/k8s-oidc-helper/)
[![Build Status](https://travis-ci.org/micahhausler/k8s-oidc-helper.svg?branch=master)](https://travis-ci.org/micahhausler/k8s-oidc-helper)

# k8s-oidc-helper

This is a small helper tool to get a user get authenticated with
[Kubernetes OIDC](http://kubernetes.io/docs/admin/authentication/) using Any OpenID Connect Provider
as the Identity Provider.

Given a ClientID, ClientSecret and Issuer URL, the tool will output the necessary
configuration for `kubectl` that you can add to `~/.kube/config`

```
$ k8s-oidc-helper -c ./client_secret.json   # Out of the Box Support for Google;s JSON File
Enter the code Google gave you: <code>

# Add the following to your ~/.kube/config
users:
- name: you@yourdomain.com
  user:
    auth-provider:
      config:
        client-id: <client-id>
        client-secret: <client-secret>
        id-token: <id-token>
        idp-issuer-url: https://accounts.google.com
        refresh-token: <refresh-token>
      name: oidc
```
Using Auth0 as your OIDC Provider
```
~/go/bin/k8s-oidc-helper --issuer-url https://your-app.auth0.com --client-id <client_id> --client-secret <client_secret>
Enter the code Provider gave you (On The page or the Value of `code` query parameter on localhost URL) : <code>
# Auth0 code sometimes ends with #, when # is not actually part of the code value itself, remove it in case you are facing errors
# Add the following to your ~/.kube/config
apiVersion: v1
clusters: []
contexts: []
current-context: ""
kind: Config
preferences: {}
users:
- name: saumitra@webklipper.com
  user:
    auth-provider:
      config:
        client-id: <client_id>
        client-secret: <client_secret>
        id-token: <id_token>
        idp-issuer-url: https://your-app.auth0.com
        refresh-token: <refresh_token>
      name: oidc
```
To merge the new configuration into your existing kubectl config file, run:

```
$ k8s-oidc-helper -c ./client_secret.json --write
Enter the code Google gave you: <code>

Configuration has been written to ~/.kube/config

# Then you can associate that user to a cluster
$ kubectl config set-context <context-name> --cluster <cluster-name> --user <you@yourdomain.com>
$ kubectl config use-context <context-name>
```

## Setup

There is a bit of setup involved before you can use this tool.

First, you'll need to create a project and OAuth 2.0 Credential in the Google
Cloud Console. You can follow [this guide](https://developers.google.com/identity/sign-in/web/devconsole-project)
on creating an application, but do *NOT* create a web application. You'll need
to select "Other" as the Application Type. Once that is created, you can
download the ClientID and ClientSecret as a JSON file for ease of use.


Second, your kube-apiserver will need the following flags on to use OpenID Connect.

```
--oidc-issuer-url=https://accounts.google.com \
--oidc-username-claim=email \       # tool supports email, sub and name claims make sure this value matches the --user-claim argument
--oidc-client-id=<Your client ID>\
```

### Role-Based Access Control

If you are using [RBAC](http://kubernetes.io/docs/admin/authorization/) as your
`--authorization-mode`, you can use the following `ClusterRole` and
`ClusterRoleBinding` for administrators that need cluster-wide access.

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1alpha1
metadata:
  name: admin-role
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
  nonResourceURLs: ["*"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1alpha1
metadata:
  name: admin-binding
subjects:
- kind: User
  name: you@yourdomain.com
roleRef:
  kind: ClusterRole
  name: admin-role
```

## Installation

```
go get github.com/micahhausler/k8s-oidc-helper
```

## Usage

```
Usage of /Users/sbhave/go/bin/k8s-oidc-helper:
      --client-id string       The ClientID for the application
      --client-secret string   The ClientSecret for the application
  -c, --config string          Path to a json file containing your Google application's ClientID and ClientSecret. Supercedes the --client-id and --client-secret flags.
      --file ~/.kube/config    The file to write to. If not specified, ~/.kube/config is used
      --issuer-url string      OIDC Discovery URL, such that <URL>/.well-known/openid-configuration can be fetched
  -o, --open                   Open the oauth approval URL in the browser (default true)
      --redirect_uri string    http://localhost or urn:ietf:wg:oauth:2.0:oob if --config flag is used for google OpenID (default "http://localhost")
      --scopes string          Required scopes to be passed to the Authicator. offline_access is added if access_type parameter is not supported by authorizer (default "openid email")
      --user-claim string      The Claim in ID-Token used to identify the user. One of sub/email/name (default "email")
  -v, --version                Print version and exit
  -w, --write                  Write config to file. Merges in the specified file
```

## License

MIT License. See [License](/LICENSE) for full text
