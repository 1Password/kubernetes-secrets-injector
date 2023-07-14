# 1Password Secrets Injector for Kubernetes

The 1Password Secrets Injector implements a mutating webhook to inject 1Password secrets as environment variables into a Kubernetes pod or deployment. Unlike the [1Password Kubernetes Operator](https://github.com/1Password/onepassword-operator), the Secrets Injector doesn't create a Kubernetes Secret when assigning secrets to your resource.

The 1Password Secrets Injector for Kubernetes can use [1Password Connect](https://developer.1password.com/docs/connect) or [1Password Service Accounts](https://developer.1password.com/docs/service-accounts) to retrieve items.

Read more on the [1Password Developer Portal](https://developer.1password.com/connect/k8s-injector).

- [Usage](#usage)
- [Setup and deployment](#setup-and-deployment)
- [Use with 1Password Connect](#use-with-1password-connect)
- [Use with 1Password Service Accounts](#use-with-1password-service-accounts)
- [Troubleshooting](#troubleshooting)
- [Security](#security)

## Usage

```yaml
# client-deployment.yaml - The client deployment/pod where you want to inject secrets

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
          # A 1Password Connect server will inject secrets into this application.
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

        - name: my-app # my-app isn't listed in the inject annotation above, so secrets won't be injected into this container.
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

<details>
<summary>Usage with 1Password Service Accounts</summary>

```yaml
# client-deployment.yaml - The client deployment/pod where you want to inject secrets

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
        operator.1password.io/version: "2-beta"
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
          # A 1Password Service Account will inject secrets into this application.
          env:
          - name: OP_SERVICE_ACCOUNT_TOKEN
            valueFrom:
              secretKeyRef:
                name: op-service-account
                key: token
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password

        - name: my-app # my-app isn't listed in the inject annotation above, so secrets won't be injected into this container.
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

</details>

To inject secrets, the container you're looking to inject into must have a `command` defined. The 1Password Secrets Injector works by mutating the `command` on init, and as such a command is needed to be mutated. If the deployments you're using aren't designed to have `command` specified in the deployment, then using the 1Password Kubernetes Operator may be a better fit.

**Note:** Injected secrets are available *only* in the current pod's session. In other words, the secrets will only be accessible for the command listed in the container specification. To access it in any other session, for example using `kubectl exec`, it's necessary to prepend `op run --` to the command.


In the example above the `app-example1` container will have injected the `DB_USERNAME` and `DB_PASSWORD` values in the session executed by the command `npm start`.

Another alternative to have the secrets available in all container's sessions is by using the [1Password Kubernetes Operator](https://github.com/1password/onepassword-operator).

## Setup and Deployment

### Prerequisites

- [docker installed](https://docs.docker.com/get-docker/)
- [kubectl installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

If you want to use 1Password Connect:

- [Set up a Secrets Automation workflow](https://developer.1password.com/docs/connect/get-started#step-1-set-up-a-secrets-automation-workflow).
- [Deploy 1Password Connect](https://developer.1password.com/docs/connect/get-started#step-2-deploy-1password-connect-server) in your Kubernetes infrastructure.

Then, follow instructions to [use the Kubernetes Injector](#use-with-1password-connect).

If you want to use 1Password Service Accounts:

- [Create a service account.](https://developer.1password.com//docs/service-accounts/)

Then, follow instructions to [use the Kubernetes Injector with a service account](#use-with-1password-service-accounts).

## Use with 1Password Connect

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

### Step 4: Annotate your client pod or deployment with `inject` annotation

Annotate your client pod or deployment spec with `operator.1password.io/inject`. It expects a comma separated list of the names of the containers that will be mutated and have secrets injected.

```yaml
# client-deployment.yaml
annotations:
  operator.1password.io/inject: "app-example1"
```

### Step 5: Configure the resource's environment

Add an environment variable to the resource with a value referencing your 1Password item. Use the following secret reference syntax: `op://<vault>/<item>[/section]/<field>`.

```yaml
env:
  - name: DB_USERNAME
    value: op://my-vault/my-item/sql/username
```

### Step 6: Provide 1Password CLI credentials on your pod or deployment

Provide your pod or deployment with 1Password CLI credentials to perform the injection. One possibility to safely provide these secrets is to [create Kubernetes Secrets](#step-1-create-a-kubernetes-secret-containing-opconnecttoken) and referring to them in your deployment configuration.

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

## Use with 1Password Service Accounts

### Step 1: Create a Kubernetes secret containing `OP_SERVICE_ACCOUNT_TOKEN`

```
kubectl create secret generic op-service-account --from-literal=token=YOUR_OP_SERVICE_ACCOUNT_TOKEN
```

### Step 2: Add the label `secrets-injection=enabled` label to the namespace

```
kubectl label namespace default secrets-injection=enabled
```

### Step 3: Deploy injector

```
make deploy
```

**NOTE:** The injector creates the TLS certificate required for the webhook to work on the fly when deploying the injector (`deployment.yaml`). When the injector is removed from the cluster, it will delete the certificate.

### Step 4: Annotate your client pod or deployment with `inject` annotation

Annotate your client pod or deployment spec with `operator.1password.io/inject`. It expects a comma separated list of the names of the containers that will be mutated and have secrets injected.

```yaml
# client-deployment.yaml
annotations:
  operator.1password.io/inject: "app-example1"
```

### Step 5: Annotate your client pod or deployment with `version` annotation

Annotate your client pod or deployment with the latest version of the 1Password CLI (`2.18.0` or later).

```yaml
# client-deployment.yaml
annotations:
  operator.1password.io/version: "2-beta"
```

### Step 6: Configure the resource's environment

Add an environment variable to the resource with a value referencing your 1Password item. Use the following secret reference syntax: `op://<vault>/<item>[/section]/<field>`.

```yaml
# client-deployment.yaml
env:
  - name: DB_USERNAME
    value: op://my-vault/my-item/sql/username
```

### Step 7: Provide 1Password CLI credentials on your pod or deployment

Provide your pod or deployment with 1Password CLI credentials to perform the injection. One possibility to safely provide these secrets is to [create Kubernetes Secrets](#step-1-create-a-kubernetes-secret-containing-opserviceaccounttoken) and referring to them in your deployment configuration.

```yaml
# client-deployment.yaml
env:
  - name: OP_SERVICE_ACCOUNT_TOKEN
    valueFrom:
      secretKeyRef:
        name: op-service-account
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

For information about our security practices, please visit the [1Password Security homepage](https://1password.com/security/).
