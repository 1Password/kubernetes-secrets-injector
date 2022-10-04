package webhook

import (
	"bytes"
	"context"
	"github.com/golang/glog"
	"k8s.io/client-go/rest"
	"os"
	"reflect"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	webhookConfigName = "secrets-injector-webhook-config"
	webhookInjectPath = "/inject"
)

type Client struct {
	Clientset kubernetes.Interface
}

var K8sClient *Client

func InitK8sClient() {
	glog.Infof("Initializing the kube client...")
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("Error creating cluster config %v", err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Errorf("Error creating clientset %v", err)
		os.Exit(1)
	}

	K8sClient = &Client{
		Clientset: clientset,
	}
}

func (c *Client) CreateOrUpdateMutatingWebhookConfiguration(caPEM *bytes.Buffer, webhookService, webhookNamespace string) error {
	glog.Infof("Creating or updating the mutatingwebhookconfiguration: %s", webhookConfigName)
	MutatingWebhookConfigV1Client := c.Clientset.AdmissionregistrationV1()
	fail := admissionregistrationv1.Fail
	sideEffect := admissionregistrationv1.SideEffectClassNone
	mutatingWebhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookConfigName,
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{{
			Name:                    "secrets-injector.1password.com",
			AdmissionReviewVersions: []string{"v1", "v1beta1"},
			SideEffects:             &sideEffect,
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				CABundle: caPEM.Bytes(), // self-generated CA for the webhook
				Service: &admissionregistrationv1.ServiceReference{
					Name:      webhookService,
					Namespace: webhookNamespace,
					Path:      &webhookInjectPath,
				},
			},
			Rules: []admissionregistrationv1.RuleWithOperations{
				{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
					},
				},
			},
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"secrets-injection": "enabled",
				},
			},
			FailurePolicy: &fail,
		}},
	}

	foundWebhookConfig, err := MutatingWebhookConfigV1Client.MutatingWebhookConfigurations().Get(context.TODO(), webhookConfigName, metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		if _, err := MutatingWebhookConfigV1Client.MutatingWebhookConfigurations().Create(context.TODO(), mutatingWebhookConfig, metav1.CreateOptions{}); err != nil {
			glog.Warningf("Failed to create the mutatingwebhookconfiguration: %s", webhookConfigName)
			return err
		}
		glog.Infof("Created mutatingwebhookconfiguration: %s", webhookConfigName)
	} else if err != nil {
		glog.Warningf("Failed to check the mutatingwebhookconfiguration: %s", webhookConfigName)
		return err
	} else {
		// there is an existing mutatingWebhookConfiguration
		if reflect.DeepEqual(foundWebhookConfig, mutatingWebhookConfig) {
			mutatingWebhookConfig.ObjectMeta.ResourceVersion = foundWebhookConfig.ObjectMeta.ResourceVersion
			if _, err := MutatingWebhookConfigV1Client.MutatingWebhookConfigurations().Update(context.TODO(), mutatingWebhookConfig, metav1.UpdateOptions{}); err != nil {
				glog.Warningf("Failed to update the mutatingwebhookconfiguration: %s", webhookConfigName)
				return err
			}
			glog.Infof("Updated the mutatingwebhookconfiguration: %s", webhookConfigName)
		}
		glog.Infof("The mutatingwebhookconfiguration: %s already exists and has no change", webhookConfigName)
	}

	return nil
}
