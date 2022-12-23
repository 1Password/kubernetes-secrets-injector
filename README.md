# 1Password Secrets Injector for Kubernetes

The 1Password Secrets Injector implements a mutating webhook to inject 1Password secrets as environment variables into a Kubernetes pod or deployment. Unlike the [1Password Kubernetes Operator](https://github.com/1Password/onepassword-operator), the Secrets Injector doesn't create a Kubernetes Secret when assigning secrets to your resource.

The 1Password Secrets Injector for Kubernetes uses 1Password Connect to retrieve items.

Read more on the [1Password Developer Portal](https://developer.1password.com/connect/k8s-injector).

- [Usage](#usage)
- [Setup and deployment](#setup-and-deployment)
- [Troubleshooting](#troubleshooting)
- [Security](#security)

## Usage
```
# client-deployment.yaml - you client deployment/pod where you want to inject secrets

apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-example
spec:
  selector:
    matchLabels:
      app: app-example
  template:
    metadata:
      annotations:
        operator.1password.io/inject: "app-example1"
      labels:
        app: app-example
    spec:
      containers:
        - name: app-example1
          image: my-image
          ports:
            - containerPort: 5000
          command: ["npm"]
          args: ["start"]
          # This app will have the secrets injected using Connect.
          env:
          - name: OP_CONNECT_HOST
            value: http://onepassword-connect:8080
          - name: OP_CONNECT_TOKEN
            valueFrom:
              secretKeyRef:
                name: connect-token
                key: token
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password

        - name: my-app # because my-app is not listed in the inject annotation above this container will not be injected with secrets
          image: my-image
          ports:
            - containerPort: 5000
          command: ["npm"]
          args: ["start"]
          env:
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password
```
**Note**: injected secrets are available in the current pod's session only.

In the example above the `app-example1` container will have injected `DB_USERNAME` and `DB_PASSWORD` values.
But if you try to access them in a new session (for example using `kubectl exec`) it would return 1password item path (aka `op://my-vault/my-item/sql/password`).

## Setup and Deployment

### Prerequisites:
- [docker installed](https://docs.docker.com/get-docker/)
- [kubectl installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [1Password Connect deployed to Kubernetes](https://developer.1password.com/docs/connect/get-started#step-2-deploy-1password-connect-server)

### Step 1: Create a Kubernetes secret containing `OP_CONNECT_TOKEN`

```shell
kubectl create secret generic connect-token --from-literal=token=YOUR_OP_CONNECT_TOKEN
```

### Step 2: Add the `secrets-injection=enabled` label to the namespace

```shell
kubectl label namespace default secrets-injection=enabled
```

### Step 3: Deploy the injector

```shell
make deploy
```
**NOTE:** The injector creates the TLS certificate required for the webhook to work on the fly when deploying the injector (`deployment.yaml`). When the injector is removed from the cluster, it will delete the certificate.


### Step 4: Annotate your client pod/deployment with `inject` annotation

Annotate your client pod/deployment specification with `operator.1password.io/inject`. It expects a comma-separated list of the containers that you want to mutate and inject secrets into.

```yaml
# client-deployment.yaml
annotations:
  operator.1password.io/inject: "app-example1"
```

### Step 5: Configure the resource's environment

Add an environment variable to the resource with a value referencing your 1Password item using secrets reference syntax: `op://<vault>/<item>[/section]/<field>`.

```yaml
env:
  - name: DB_USERNAME
    value: op://my-vault/my-item/sql/username
```

### Step 6: Provide 1Password CLI credentials on your pod/deployment

You can provide your pod or deployment with 1Password CLI credentials by [creating Kubernetes Secrets](#step-1--create-a-kubernetes-secret-containing-opconnecttoken) and referring to them in your deployment configuration.
```yaml
# your-app-pod/deployment.yaml
env:
  - name: OP_CONNECT_HOST
    value: http://onepassword-connect:8080
  - name: OP_CONNECT_TOKEN
    valueFrom:
      secretKeyRef:
        name: connect-token
        key: token
  - name: DB_USERNAME
    value: op://my-vault/my-item/sql/username
```

## Troubleshooting

If you can't inject secrets in your pod, make sure:
- The namespace of your pod has the `secrets-injection=enabled` label
- The 1Password Secret Injector webhook is running (`secrets-injector` by default).
- Your container has a `command` field specifying the command to run the app in your container

## Security

1Password requests you practice responsible disclosure if you discover a vulnerability.

Please file requests through [**BugCrowd**](https://bugcrowd.com/agilebits)

For information about our security practices, please visit our [Security homepage](https://1password.com/security/).
