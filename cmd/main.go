package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/1password/kubernetes-secrets-injector/pkg/webhook"
	"github.com/golang/glog"
)

var (
	webhookNamespace, webhookServiceName string
)

func init() {
	// webhook server running namespace
	webhookNamespace = os.Getenv("POD_NAMESPACE")
}

func main() {
	webhook.InitK8sClient()

	var parameters webhook.SecretInjectorParameters

	glog.Info("Starting webhook")
	// get command line parameters
	flag.IntVar(&parameters.Port, "port", 8443, "Webhook server port.")
	flag.StringVar(&webhookServiceName, "service-name", "secrets-injector-svc", "Webhook service name.")
	flag.Parse()

	dnsNames := []string{
		webhookServiceName,
		webhookServiceName + "." + webhookNamespace,
		webhookServiceName + "." + webhookNamespace + ".svc",
	}
	commonName := webhookServiceName + "." + webhookNamespace + ".svc"

	org := "1password.com"

	caPEM, certPEM, certKeyPEM, err := generateCert([]string{org}, dnsNames, commonName)
	if err != nil {
		glog.Errorf("Failed to generate ca and certificate key pair: %v", err)
		os.Exit(1)
	}

	pair, err := tls.X509KeyPair(certPEM.Bytes(), certKeyPEM.Bytes())
	if err != nil {
		glog.Errorf("Failed to load key pair: %v", err)
		os.Exit(1)
	}

	// create or update the mutatingwebhookconfiguration
	err = webhook.K8sClient.CreateOrUpdateMutatingWebhookConfiguration(caPEM, webhookServiceName, webhookNamespace)
	if err != nil {
		glog.Errorf("Failed to create or update the mutating webhook configuration: %v", err)
		os.Exit(1)
	}

	secretInjector := &webhook.SecretInjector{
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", parameters.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/inject", secretInjector.Serve)
	secretInjector.Server.Handler = mux

	// start webhook server in new routine
	go func() {
		if err := secretInjector.Server.ListenAndServeTLS("", ""); err != nil {
			glog.Errorf("Failed to listen and serve webhook server: %v", err)
			os.Exit(1)
		}
	}()

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	glog.Infof("Got OS shutdown signal, shutting down webhook server gracefully...")
	secretInjector.Server.Shutdown(context.Background())
}
