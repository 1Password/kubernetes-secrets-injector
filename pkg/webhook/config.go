package webhook

import (
	"os"

	"github.com/golang/glog"
)

const (
	connectTokenSecretKeyEnv    = "OP_CONNECT_TOKEN_KEY"
	connectTokenSecretNameEnv   = "OP_CONNECT_TOKEN_NAME"
	connectHostEnv              = "OP_CONNECT_HOST"
	connectTokenEnv             = "OP_CONNECT_TOKEN"
	serviceAccountSecretKeyEnv  = "OP_SERVICE_ACCOUNT_TOKEN_KEY"
	serviceAccountSecretNameEnv = "OP_SERVICE_ACCOUNT_SECRET_NAME"
)

type Config struct {
	Connect        *ConnectConfig
	ServiceAccount *ServiceAccountConfig
}

type ServiceAccountConfig struct {
	SecretName string // the name of the secret that stores the service account token
	TokenKey   string // the name of the data field in the secret the stores the service account token
}

type ConnectConfig struct {
	Host       string // the host in which a connect server is running
	SecretName string // the token name of the secret that stores the connect token
	TokenKey   string // the name of the data field in the secret the stores the connect token
}

func CreateWebhookConfig() Config {
	serviceAccountConfig := createServiceAccountConfig(serviceAccountSecretNameEnv, serviceAccountSecretKeyEnv)
	connectConfig := createConnectConfig(connectHostEnv, connectTokenSecretNameEnv, connectTokenSecretKeyEnv)

	if serviceAccountConfig == nil && connectConfig == nil {
		glog.Error("Error creating webhook config. Provide valid OP_CONNECT_* or OP_SERVICE_ACCOUNT_* env variables.")
		os.Exit(1)
	}

	return Config{
		ServiceAccount: serviceAccountConfig,
		Connect:        connectConfig,
	}
}

func createServiceAccountConfig(secretName string, dataKey string) *ServiceAccountConfig {
	serviceAccountTokenSecretName, present := os.LookupEnv(secretName)
	if !present || serviceAccountTokenSecretName == "" {
		glog.Info("Service Account secret not set")
		return nil
	}

	serviceAccountTokenKey, present := os.LookupEnv(dataKey)
	if !present || serviceAccountTokenKey == "" {
		glog.Info("Service Account secret key not set")
		return nil
	}

	glog.Info("Service Account config is set")

	return &ServiceAccountConfig{
		SecretName: serviceAccountTokenSecretName,
		TokenKey:   serviceAccountTokenKey,
	}
}

func createConnectConfig(host string, secretName string, dataKey string) *ConnectConfig {
	connectHost, present := os.LookupEnv(host)
	if !present || connectHost == "" {
		glog.Error("Connect host not set")
		return nil
	}

	connectTokenName, present := os.LookupEnv(secretName)
	if !present || connectTokenName == "" {
		glog.Error("Connect token name not set")
		return nil
	}

	connectTokenKey, present := os.LookupEnv(dataKey)
	if !present || connectTokenKey == "" {
		glog.Error("Connect token key not set")
		return nil
	}

	glog.Info("Connect config is set")

	return &ConnectConfig{
		Host:       connectHost,
		SecretName: connectTokenName,
		TokenKey:   connectTokenKey,
	}
}
