package webhook

import (
	"os"

	"github.com/golang/glog"
)

const (
	connectTokenSecretKeyEnv    = "OP_CONNECT_TOKEN_KEY"
	connectTokenSecretNameEnv   = "OP_CONNECT_TOKEN_NAME"
	connectHostEnv              = "OP_CONNECT_HOST"
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
	Host      string // the host in which a connect server is running
	TokenName string // the token name of the secret that stores the connect token
	TokenKey  string // the name of the data field in the secret the stores the connect token
}

func CreateWebhookConfig() Config {
	var config Config

	if serviceAccountConfig := createServiceAccountConfig(serviceAccountSecretNameEnv, serviceAccountSecretKeyEnv); serviceAccountConfig != nil {
		config = Config{
			ServiceAccount: serviceAccountConfig,
		}
		glog.Info("Using Service Account integration")
	} else {
		connectConfig := createConnectConfig(connectHostEnv, connectTokenSecretNameEnv, connectTokenSecretKeyEnv)
		config = Config{
			Connect: connectConfig,
		}
		glog.Info("Using with Connect integration")
	}

	return config
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

	return &ServiceAccountConfig{
		SecretName: serviceAccountTokenSecretName,
		TokenKey:   serviceAccountTokenKey,
	}
}

func createConnectConfig(host string, secretName string, dataKey string) *ConnectConfig {
	connectHost, present := os.LookupEnv(connectHostEnv)
	if !present || connectHost == "" {
		glog.Error("Connect host not set")
	}

	connectTokenName, present := os.LookupEnv(secretName)
	if !present || connectTokenName == "" {
		glog.Error("Connect token name not set")
	}

	connectTokenKey, present := os.LookupEnv(dataKey)
	if !present || connectTokenKey == "" {
		glog.Error("Connect token key not set")
	}

	return &ConnectConfig{
		Host:      connectHost,
		TokenName: connectTokenName,
		TokenKey:  connectTokenKey,
	}
}
