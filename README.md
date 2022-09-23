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

The 1Password Secrets Injector for Kubernetes can use 1Password Connect or Service Account to retrieve items.
Service Accounts are currently in Beta and are only available to select users.

### Prerequisites:

- [docker installed](https://docs.docker.com/get-docker/)
- [kubectl installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Follow the [Using with 1Password Connect guide](#using-1password-connect) if you want to go with 1Password Connect, or [Using with Service Account guide](#using-with-service-account) if you want to go with Service Account.

If you setup injector to use Connect and Service Account together. The Connect will take preference.

## Using with 1Password Connect

### 1. Setup and deploy 1Password Connect

You should deploy 1Password Connect to your infrastructure in order to retrieve items from 1Password.

- [setup 1Password Connect server](https://developer.1password.com/docs/connect/get-started#step-1-set-up-a-secrets-automation-workflow)
- [deploy 1Password Connect to Kubernetes](https://developer.1password.com/docs/connect/get-started#step-2-deploy-1password-connect-server)

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

## Using with Service Account

**_ Note: _** Service Accounts are currently in Beta and are only available to select users.

### 1. Create kubernetes secret containing `OP_SERVICE_ACCOUNT_TOKEN`

**_ Note: _** Replace OP_SERVICE_ACCOUNT_SECRET_NAME, OP_SERVICE_ACCOUNT_TOKEN_KEY, YOUR_OP_SERVICE_ACCOUNT_TOKEN with values you'd like to use.

- `OP_SERVICE_ACCOUNT_SECRET_NAME` - name of the secret that stores the service account token.
- `OP_SERVICE_ACCOUNT_TOKEN_KEY` - name of the data field in the secret that stores the service account token
- `YOUR_OP_SERVICE_ACCOUNT_TOKEN` - your Service Acccount token

```
kubectl create secret generic OP_SERVICE_ACCOUNT_SECRET_NAME --from-literal=OP_SERVICE_ACCOUNT_TOKEN_KEY=YOUR_OP_SERVICE_ACCOUNT_TOKEN -n op-injector
```

### 2.Deploy injector

```
kubectl create -f deploy/permissions.yaml
kubectl create -f deploy/deployment.yaml
kubectl create -f deploy/service.yaml
```

**NOTE:** The injector creates the TLS certificate required for the webhook to work on the fly when deploying the injector (`deployment.yaml`). Also, the injector will delete the certificate when the injector is removed from the cluster.

## Troubleshooting

If you are trouble getting secrets injected in your pod, check the following:

1. Check that that the namespace of your pod has the `op-secret-injection=enabled` label
2. Check that the `caBundle` in `mutatingwebhook.yaml` is set with a correct value
3. Ensure that the 1Password Secret Injector webhook is running (`op-injector` by default).
4. Check that your container has a `command` field specifying the command to run the app in your container
