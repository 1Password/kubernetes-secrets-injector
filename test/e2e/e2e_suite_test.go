package e2e

import (
	"context"
	"path/filepath"

	"github.com/1password/kubernetes-secrets-injector/pkg/testhelper/defaults"
	"github.com/1password/kubernetes-secrets-injector/pkg/testhelper/kind"
	"github.com/1password/kubernetes-secrets-injector/pkg/testhelper/kube"
	"github.com/1password/kubernetes-secrets-injector/pkg/testhelper/system"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	imageName     = "1password/kubernetes-secrets-injector:latest"
	containerName = "app-example"
)

var kubeClient *kube.Kube

var _ = Describe("Kubernetes Secrets Injector e2e", Ordered, func() {
	ctx := context.Background()

	BeforeAll(func() {
		kubeClient = kube.NewKubeClient(&kube.Config{
			Namespace:    "default",
			ManifestsDir: filepath.Join("manifests"),
			TestConfig: &kube.TestConfig{
				Timeout:  defaults.E2ETimeout,
				Interval: defaults.E2EInterval,
			},
		})

		By("Building injector image")
		_, err := system.Run("make", "docker-build")
		Expect(err).NotTo(HaveOccurred())

		By("Building" + containerName + "image")
		root, err := system.GetProjectRoot()
		Expect(err).NotTo(HaveOccurred())
		clientDir := filepath.Join(root, "test", "e2e", "client")
		_, err = system.Run("docker", "build", "-t", containerName+":latest", clientDir)
		Expect(err).NotTo(HaveOccurred())

		kind.LoadImageToKind(imageName)
		kind.LoadImageToKind(containerName + ":latest")

		kubeClient.Secret("op-credentials").CreateOpCredentials(ctx)
		kubeClient.Secret("op-credentials").CheckIfExists(ctx)

		kubeClient.Secret("onepassword-token").CreateFromEnvVar(ctx, "OP_CONNECT_TOKEN")
		kubeClient.Secret("onepassword-token").CheckIfExists(ctx)

		kubeClient.Secret("onepassword-service-account-token").CreateFromEnvVar(ctx, "OP_SERVICE_ACCOUNT_TOKEN")
		kubeClient.Secret("onepassword-service-account-token").CheckIfExists(ctx)

		kubeClient.Namespace("default").LabelNamespace(ctx, map[string]string{
			"secrets-injection": "enabled",
		})

		_, err = system.Run("make", "deploy")
		Expect(err).NotTo(HaveOccurred())

		kubeClient.Pod(map[string]string{"app": "secrets-injector"}).WaitingForRunningPod(ctx)

		//time.Sleep(5 * time.Second)
		kubeClient.Webhook("secrets-injector-webhook-config").WaitForWebhookToBeRegistered(ctx)

		By("Deploy test app")
		yamlPath := filepath.Join(root, "test", "e2e", "manifests", "client.yaml")
		_, err = system.Run("kubectl", "apply", "-f", yamlPath)
		Expect(err).NotTo(HaveOccurred())

		kubeClient.Pod(map[string]string{"app": containerName}).WaitingForRunningPod(ctx)
	})

	//Context("Use Injector with Connect", func() {
	//	BeforeAll(func() {
	//		kubeClient.Pod(map[string]string{"app": "onepassword-connect"}).WaitingForRunningPod(ctx)
	//	})
	//
	//	runCommonTestCases(ctx)
	//})

	Context("Use the operator with Service Account", func() {
		//BeforeAll(func() {
		//	// Update test-app to use Service Account
		//	kubeClient.Deployment("test-app").PatchEnvVars(ctx, []corev1.EnvVar{
		//		{
		//			Name: "OP_SERVICE_ACCOUNT_TOKEN",
		//			ValueFrom: &corev1.EnvVarSource{
		//				SecretKeyRef: &corev1.SecretKeySelector{
		//					LocalObjectReference: corev1.LocalObjectReference{
		//						Name: "onepassword-service-account-token",
		//					},
		//					Key: "token",
		//				},
		//			},
		//		},
		//	}, []string{"OP_CONNECT_HOST", "OP_CONNECT_TOKEN"})
		//})

		runCommonTestCases(ctx)
	})
})

// runCommonTestCases contains test cases that are common to both Connect and Service Account authentication methods.
func runCommonTestCases(ctx context.Context) {
	It("Should NOT inject env variables into test app container without annotation", func() {
		kubeClient.Pod(map[string]string{
			"app": containerName,
		}).VerifySecretsNotInjected(ctx)
	})

	It("Should inject the webhook and secret into app pod", func() {
		kubeClient.Deployment("app-example").AddAnnotation(ctx, map[string]string{
			"operator.1password.io/inject": containerName,
		})

		kubeClient.Pod(map[string]string{
			"app": containerName,
		}).VerifyWebhookInjection(ctx)

		kubeClient.Pod(map[string]string{
			"app": containerName,
		}).VerifySecretsInjected(ctx)
	})
}
