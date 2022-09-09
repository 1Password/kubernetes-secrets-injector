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

### 2. Secure pod connection to inject secrets

[Follow this guide](#secure-pod-connection-to-inject-secrets)

### 3. Create kubernetes secret containing `OP_CONNECT_TOKEN`

```
kubectl create secret generic OP_CONNECT_TOKEN_NAME --from-literal=OP_CONNECT_TOKEN_KEY=YOUR_OP_CONNECT_TOKEN -n op-injector
```

**_ Note: _** Replace OP_CONNECT_TOKEN_NAME, OP_CONNECT_TOKEN_KEY, YOUR_OP_CONNECT_TOKEN with values you'd like to use.

- `OP_CONNECT_TOKEN_NAME` - name of the secret that stores the connect token
- `OP_CONNECT_TOKEN_KEY` - name of the data field in the secret the stores the connect token
- `YOUR_OP_CONNECT_TOKEN` - your 1Password Connect token

### 4.Deploy injector
Copy and run the next scripts from `deploy` folder specifying `OP_CONNECT_HOST`, `OP_CONNECT_TOKEN_NAME` and `OP_CONNECT_TOKEN_KEY` env variables. They should equal to those you set in the [step 3](#3-create-kubernetes-secret-containing-opconnecttoken)

```
kubectl create -f deploy/deployment.yaml
kubectl create -f deploy/service.yaml
kubectl create -f deploy/mutatingwebhook.yaml
```

## Using with Service Account

**_ Note: _** Service Accounts are currently in Beta and are only available to select users.

### 1. [Secure pod connection to inject secrets](#secure-pod-connection-to-inject-secrets)

### 2. Create kubernetes secret containing `OP_SERVICE_ACCOUNT_TOKEN`

**_ Note: _** Replace OP_SERVICE_ACCOUNT_SECRET_NAME, OP_SERVICE_ACCOUNT_TOKEN_KEY, YOUR_OP_SERVICE_ACCOUNT_TOKEN with values you'd like to use.

- `OP_SERVICE_ACCOUNT_SECRET_NAME` - name of the secret that stores the service account token.
- `OP_SERVICE_ACCOUNT_TOKEN_KEY` - name of the data field in the secret that stores the service account token
- `YOUR_OP_SERVICE_ACCOUNT_TOKEN` - your Service Acccount token

```
kubectl create secret generic OP_SERVICE_ACCOUNT_SECRET_NAME --from-literal=OP_SERVICE_ACCOUNT_TOKEN_KEY=YOUR_OP_SERVICE_ACCOUNT_TOKEN -n op-injector
```

### 3. Deploy injector

Copy and run the next scripts from `deploy` folder specifying `OP_SERVICE_ACCOUNT_SECRET_NAME` and `OP_SERVICE_ACCOUNT_TOKEN_KEY` env variables. They should equal to those you set in the [step 2](#2-create-kubernetes-secret-containing-opserviceaccounttoken)

```
kubectl create -f deploy/deployment.yaml
kubectl create -f deploy/service.yaml
kubectl create -f deploy/mutatingwebhook.yaml
```

### 5. Inject 1Password Item into your pod

## Secure pod connection to inject secrets

The 1Password Secrets Injector for Kubernetes uses a webhook server in order to inject secrets into pods and deployments. Admission to the webhook server must be a secure operation, thus communication with the webhook server requires a TLS certificate signed by a Kubernetes CA.

For managing TLS certifcates for your cluster please see the [official documentation](https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/). The certificate and key generated in the offical documentation must be set in the [deployment](deploy/deployment.yaml) arguments (`tlsCertFile` and `tlsKeyFile` respectively) for the Secret injector.

In additon to setting the tlsCert and tlsKey for the Secret Injector service, we must also create a webhook configuration for the service. An example of the confiugration can be found [here](deploy/mutatingwebhook.yaml). In the provided example you may notice that the caBundle is not set. Please replace this value with your caBundle. This can be generated with the Kubernetes apiserver's default caBundle with the following command

`export CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')`

```
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: op-injector-webhook-config
  labels:
    app: op-injector
webhooks:
  - name: operator.1password.io
    failurePolicy: Fail
    clientConfig:
      service:
        name: op-injector-svc
        namespace: op-injector
        path: "/inject"
      caBundle: ${CA_BUNDLE} //replace this with your own CA Bundle
    admissionReviewVersions: ["v1", "v1beta1"]
    sideEffects: None
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: [""]
        apiVersions: ["v1"]
        resources: ["pods"]
    namespaceSelector:
      matchLabels:
        op-secret-injection: enabled
```

## Troubleshooting

If you are trouble getting secrets injected in your pod, check the following:

1. Check that that the namespace of your pod has the `op-secret-injection=enabled` label
2. Check that the `caBundle` in `mutatingwebhook.yaml` is set with a correct value
3. Ensure that the 1Password Secret Injector webhook is running (`op-injector` by default).
4. Check that your container has a `command` field specifying the command to run the app in your container
