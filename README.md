# 1Password Secrets Injector for Kubernetes

The 1Password Secrets Injector implements a mutating webhook to inject 1Password secrets as environment variables into a pod or deployment. Unlike the 1Password Kubernetes Operator, the Secrets Injector does not create a Kubernetes Secret when assigning secrets to your resource.

- [Example](#example)
- [Setup and deployment](#setup-and-deployment)
- [Use with 1Password Connect](#use-with-1password-connect)
- [Use with Service Account](#use-with-service-account)
- [Use with the 1Password Kubernetes Operator](#use-with-the-1password-kubernetes-operator)
- [Provide `op-cli` credentials on your app pod/deployment](#provide-op-cli-credentials-on-your-app-poddeployment)
- [Troubleshooting](#troubleshooting)

## Example
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
        operator.1password.io/inject: "app-example1,app-example2,app-example3"
      labels:
        app: app-example
    spec:
      containers:
        - name: app-example
          image: my-image
          command: ["./example"]
          # as OP_CONNECT_TOKEN or OP_SERVICE_ACCOUNT_TOKEN no provided,
          # it tries to extract from default secrets
          # OP_CONNECT_TOKEN from the secret `connect-token` with the key `token`
          # OP_SERVICE_ACCOUT_TOKEN from the secret `service-account` with the key `token`
          env:
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password
        
        - name: app-example2
          image: my-image
          # it tries to find OP_CONNECT_TOKEN and OP_SERVICE_ACCOUNT_TOKEN 
          # from the provided secret with provided token
          # OP_CONNECT_TOKEN - from secret with the name OP_CONNECT_TOKEN_SECRET_NAME and
          #                     data key OP_CONNECT_TOKEN_KEY 
          # OP_SERVICE_ACCOUNT_TOKEN - from secret with the name OP_SERVICE_ACCOUNT_SECRET_NAME and
          #                            data key OP_SERVICE_ACCOUNT_TOKEN_KEY 
          env:
          - name: OP_CONNECT_TOKEN_SECRET_NAME
            value: connect-token
          - name: OP_CONNECT_TOKEN_KEY
            value: token
          - name: OP_SERVICE_ACCOUNT_SECRET_NAME
            value: service-account
          - name: OP_SERVICE_ACCOUNT_TOKEN_KEY
            value: token
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password
            
        - name: app-example2
          image: my-image
          # it tries to find OP_CONNECT_TOKEN and OP_SERVICE_ACCOUNT_TOKEN 
          # from the provided secrets
          env:
          - name: OP_CONNECT_TOKEN
            valueFrom:
              secretKeyRef:
                name: connect-token
                key: token
          - name: OP_SERVICE_ACCOUNT_TOKEN
            valueFrom:
              secretKeyRef:
                name: service-account
                key: token
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

## Setup and Deployment
The 1Password Secrets Injector for Kubernetes can use 1Password Connect or Service Account to retrieve items.
Service Accounts are currently in Beta and are only available to select users.

### Prerequisites:
- [docker installed](https://docs.docker.com/get-docker/)
- [kubectl installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Follow the [Use with 1Password Connect guide](#use-with-1password-connect) if you want to go with 1Password Connect, or [Use with Service Account guide](#use-with-service-account) if you want to go with Service Account.

If you set up injector to use Connect and Service Account together. The Connect will take preference.


## Use with 1Password Connect
### 1. Setup and deploy 1Password Connect

You should deploy 1Password Connect to your infrastructure in order to retrieve items from 1Password.

- [setup 1Password Connect server](https://developer.1password.com/docs/connect/get-started#step-1-set-up-a-secrets-automation-workflow)
- [deploy 1Password Connect to Kubernetes](https://developer.1password.com/docs/connect/get-started#step-2-deploy-1password-connect-server)

### 2. Add the label `secrets-injection=enabled` label to the namespace:
```
kubectl label namespace default secrets-injection=enabled
```

### 3. Create kubernetes secret containing `OP_CONNECT_TOKEN`
**_ Note: _** Replace OP_CONNECT_TOKEN_SECRET_NAME, OP_CONNECT_TOKEN_KEY, YOUR_OP_CONNECT_TOKEN with values you'd like to use.

- `OP_CONNECT_TOKEN_SECRET_NAME` - name of the secret that stores the Connect token.
- `OP_CONNECT_TOKEN_KEY` - name of the data field in the secret that stores the Connect token
- `YOUR_OP_CONNECT_TOKEN` - your Connect token
```
kubectl create secret generic OP_CONNECT_TOKEN_SECRET_NAME --from-literal=OP_CONNECT_TOKEN_KEY=YOUR_OP_CONNECT_TOKEN
```

### 4.Deploy injector
```
kubectl create -f deploy/permissions.yaml
kubectl create -f deploy/deployment.yaml
kubectl create -f deploy/service.yaml
```
**NOTE:** The injector creates the TLS certificate required for the webhook to work on the fly when deploying the injector (`deployment.yaml`). Also, the injector will delete the certificate when the injector is removed from the cluster.


### 5. Annotate your client pod/deployment spec with `operator.1password.io/inject` which expects a comma separated list of the names of the containers to that will be mutated and have secrets injected.
```
# client-deployment.yaml
annotations:
  operator.1password.io/inject: "app-example1,app-example2,app-example3"
```

### 6. Add an environment variable to the resource with a value referencing your 1Password item in the format `op://<vault>/<item>[/section]/<field>`.
```
env:
  - name: DB_USERNAME
    value: op://my-vault/my-item/sql/username
```

### 7. [Provide op-cli credentials on your app pod/deployment](#provide-op-cli-credentials-on-your-app-poddeployment)


## Use with Service Account
**_ Note: _** Service Accounts are currently in Beta and are only available to select users.

### 1. Create kubernetes secret containing `OP_SERVICE_ACCOUNT_TOKEN`
**_ Note: _** Replace OP_SERVICE_ACCOUNT_SECRET_NAME, OP_SERVICE_ACCOUNT_TOKEN_KEY, YOUR_OP_SERVICE_ACCOUNT_TOKEN with values you'd like to use.

- `OP_SERVICE_ACCOUNT_SECRET_NAME` - name of the secret that stores the Service Account token.
- `OP_SERVICE_ACCOUNT_TOKEN_KEY` - name of the data field in the secret that stores the Service Account token
- `YOUR_OP_SERVICE_ACCOUNT_TOKEN` - your Service Account token

```
kubectl create secret generic OP_SERVICE_ACCOUNT_SECRET_NAME --from-literal=OP_SERVICE_ACCOUNT_TOKEN_KEY=YOUR_OP_SERVICE_ACCOUNT_TOKEN
```

### 2. Add the label `secrets-injection=enabled` label to the namespace:
```
kubectl label namespace default secrets-injection=enabled
```

### 3.Deploy injector
```
kubectl create -f deploy/permissions.yaml
kubectl create -f deploy/deployment.yaml
kubectl create -f deploy/service.yaml
```
**NOTE:** The injector creates the TLS certificate required for the webhook to work on the fly when deploying the injector (`deployment.yaml`). Also, the injector will delete the certificate when the injector is removed from the cluster.

### 4. Annotate your client pod/deployment spec with `operator.1password.io/inject` which expects a comma separated list of the names of the containers to that will be mutated and have secrets injected.
```
# client-deployment.yaml
annotations:
  operator.1password.io/inject: "app-example1,app-example2,app-example3"
```

### 5. Annotate your client pod/deployment with the minimum op-cli version  annotation that supports Service Accounts `2.7.1-beta.01`
```
# client-deployment.yaml
annotations:
  operator.1password.io/version: "2.7.1-beta.01"
```

### 5. Add an environment variable to the resource with a value referencing your 1Password item in the format `op://<vault>/<item>[/section]/<field>`.
```
env:
  - name: DB_USERNAME
    value: op://my-vault/my-item/sql/username
```

### 6. [Provide op-cli credentials on your app pod/deployment](#provide-op-cli-credentials-on-your-app-poddeployment)


## Use with the 1Password Kubernetes Operator
The 1Password Secrets Injector for Kubernetes can be used in conjuction with the 1Password Kubernetes Operator in order to provide automatic deployment restarts when a 1Password item being used by your deployment has been updated.

[Click here for more details on the 1Password Kubernetes Operator](https://github.com/1Password/onepassword-operator)


## Provide `op-cli` credentials on your app pod/deployment
**_ Note: _** `OP_CONNECT_HOST` default `http://onepassword-connect:8080` if not set explicitly
#### Do not forget to create secrets containing op-cli tokens

You can do that in the different ways:

1. Use default values to extract from the secret
```
kubectl create secret generic connect-token --from-literal=token=YOUR_TOKEN
kubectl create secret generic service-account --from-literal=token=YOUR_TOKEN

# your-app-pod/deployment.yaml
env:
  # OP_CONNECT_HOST default value is 'http://onepassword-connect:8080'
  # OP_CONNECT_TOKEN from the secret `connect-token` with the key `token`
  # OP_SERVICE_ACCOUT_TOKEN from the secret `service-account` with the key `token`
  - name: DB_USERNAME
    value: op://my-vault/my-item/sql/username
```
2. Directly set env variables `OP_CONNECT_HOST`, `OP_CONNECT_TOKEN`, `OP_SERVICE_ACCOUNT_TOKEN`
```
- env:
  - name: OP_CONNECT_HOST
    value: http://onepassword-connect:8080
  - name: OP_CONNECT_TOKEN
    value: abcd...abcd
  - name: OP_SERVICE_ACCOUNT_TOKEN
    value: abcd...abcd
  - name: DB_USERNAME
    value: op://my-vault/my-item/sql/username
```
3. As the reference to the secret
```
kubectl create secret generic connect-token --from-literal=token=YOUR_TOKEN
kubectl create secret generic service-account --from-literal=token=YOUR_TOKEN

# your-app-pod/deployment.yaml
env:
  # OP_CONNECT_HOST default value is 'http://onepassword-connect:8080'
  - name: OP_CONNECT_TOKEN
    valueFrom:
      secretKeyRef:
        name: connect-token
        key: token
  - name: OP_SERVICE_ACCOUNT_TOKEN
    valueFrom:
      secretKeyRef:
        name: service-account
        key: token
  - name: DB_USERNAME
    value: op://my-vault/my-item/sql/username
```
4. As the reference to the secret using env variables
```
kubectl create secret generic connect-token --from-literal=token=YOUR_TOKEN
kubectl create secret generic service-account --from-literal=token=YOUR_TOKEN

# your-app-pod/deployment.yaml
env:
  # OP_CONNECT_HOST default value is 'http://onepassword-connect:8080'
  - name: OP_CONNECT_TOKEN_SECRET_NAME
    value: connect-token
  - name: OP_CONNECT_TOKEN_KEY
    value: token
  - name: OP_SERVICE_ACCOUNT_SECRET_NAME
    value: service-account
  - name: OP_SERVICE_ACCOUNT_TOKEN_KEY
    value: token
  - name: DB_USERNAME
    value: op://my-vault/my-item/sql/username
```


## Troubleshooting
If you are trouble getting secrets injected in your pod, check the following:

1. Check that the namespace of your pod has the `secrets-injection=enabled` label
2. Ensure that the 1Password Secret Injector webhook is running (`secrets-injector` by default).
3. Check that your container has a `command` field specifying the command to run the app in your container
