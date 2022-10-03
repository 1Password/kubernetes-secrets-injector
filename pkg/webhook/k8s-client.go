package webhook

import (
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	admissionregistrationv1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"os"
)

var (
	MutatingWebhookConfigV1Client admissionregistrationv1.AdmissionregistrationV1Interface
	CoreV1Client                  corev1.CoreV1Interface
)

func init() {
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

	MutatingWebhookConfigV1Client = clientset.AdmissionregistrationV1()
	CoreV1Client = clientset.CoreV1()
}
