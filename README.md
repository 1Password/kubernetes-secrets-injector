# 1Password Secrets Injector for Kubernetes

The 1Password Secrets Injector implements a mutating webhook to inject 1Password secrets as environment variables into a pod or deployment. Unlike the 1Password Kubernetes Operator, the Secrets Injector does not create a Kubernetes Secret when assigning secrets to your resource.

## Usage

[Setup the infrastructure](#setup-and-deployment)

For every namespace you want the 1Password Secret Injector to inject secrets for, you must add the label `op-secret-injection=enabled` label to the namespace:

```
kubectl label namespace <namespace> op-secret-injection=enabled
```

To inject a 1Password secret as an environment variable, your pod or deployment you must add an environment variable to the resource with a value referencing your 1Password item in the format `op://<vault>/<item>[/section]/<field>`. You must also annotate your pod/deployment spec with `operator.1password.io/inject` which expects a comma separated list of the names of the containers to that will be mutated and have secrets injected.

Note: You must also include the command needed to run the container as the secret injector prepends a script to this command in order to allow for secret injection.

```
#example

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
        operator.1password.io/inject: "app-example,another-example"
      labels:
        app: app-example
    spec:
      containers:
        - name: app-example
          image: my-image
          command: ["./example"]
          env:
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password
        - name: another-example
          image: my-image
          env:
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password
        - name: my-app //because my-app is not listed in the inject annotation above this container will not be injected with secrets
          image: my-image
          env:
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password
```

## Use with the 1Password Kubernetes Operator

The 1Password Secrets Injector for Kubernetes can be used in conjuction with the 1Password Kubernetes Operator in order to provide automatic deployment restarts when a 1Password item being used by your deployment has been updated.

[Click here for more details on the 1Password Kubernetes Operator](https://github.com/1Password/onepassword-operator)

## Setup and Deployment

### Prerequisites:

- [docker installed](https://docs.docker.com/get-docker/)
- [kubectl installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [1Password Connect server configured](https://developer.1password.com/docs/connect/get-started#step-1-set-up-a-secrets-automation-workflow)
- [1Password Connect deployed to Kubernetes](https://developer.1password.com/docs/connect/get-started#step-2-deploy-1password-connect-server)

### 1. Setup and deploy 1Password Connect

The 1Password Secrets Injector for Kubernetes uses 1Password Connect to retrieve items. You should deploy 1Password Connect to your infrastructure. Please see [Prerequisites section](#prerequisites) to do that.

### 2. Create kubernetes secret containing `OP_CONNECT_TOKEN`

```
kubectl create secret generic onepassword-token --from-literal=token=YOUR_OP_CONNECT_TOKEN -n op-injector
```

### 3.Deploy injector

```
kubectl create -f deploy/permissions.yaml
kubectl create -f deploy/deployment.yaml
kubectl create -f deploy/service.yaml
```

**NOTE:** The injector creates the TLS certificate required for the webhook to work on the fly when deploying the injector (`deployment.yaml`). Also, the injector will delete the certificate when the injector is removed from the cluster.

## Troubleshooting

If you are trouble getting secrets injected in your pod, check the following:

1. Check that that the namespace of your pod has the `op-secret-injection=enabled` label
2. Check that the `caBundle` in `mutatingwebhook-ca-bundle.yaml` is set with a correct value
3. Ensure that the 1Password Secret Injector webhook is running (`op-injector` by default).
4. Check that your container has a `command` field specifying the command to run the app in your container
