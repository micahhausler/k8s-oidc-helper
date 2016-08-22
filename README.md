# k8s-oidc-helper

This is a small helper tool to get a user get authenticated with
[Kubernetes OIDC](http://kubernetes.io/docs/admin/authentication/) using Google
as the Identity Provider.

Given a ClientID and ClientSecret, the tool will output the necessary
configuration for `kubectl` that you can add to `~/.kube/config`

```
$ k8s-oidc-helper -c ./client_secret.json
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
--oidc-username-claim=email \
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
Usage of k8s-oidc-helper:

  --client-id string
        The ClientID for the application
  --client-secret string
        The ClientSecret for the application
  -c, --config string
        Path to a json file containing your application's ClientID and ClientSecret. Supercedes the --client-id and --client-secret flags.
  -o, --open true
        Open the oauth approval URL in the browser. Defaults to true (default true)
  -v, --version
        print version and exit
```

## Wishlist

- [ ] Add tests/CI
- [ ] Add docker builds to CI

## License

MIT License. See [License](/LICENSE) for full text
