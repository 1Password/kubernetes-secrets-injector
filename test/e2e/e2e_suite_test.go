package e2e

import (
	"context"
	"path/filepath"

	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/pkg/testhelper/defaults"
	"github.com/1Password/onepassword-operator/pkg/testhelper/kind"
	"github.com/1Password/onepassword-operator/pkg/testhelper/kube"
	"github.com/1Password/onepassword-operator/pkg/testhelper/system"
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

		By("Building client image")
		root, err := system.GetProjectRoot()
		Expect(err).NotTo(HaveOccurred())
		clientDir := filepath.Join(root, "test", "e2e", "client")
		_, err = system.Run("docker", "build", "-t", containerName+":latest", clientDir)
		Expect(err).NotTo(HaveOccurred())

		kind.LoadImageToKind(imageName)
		kind.LoadImageToKind(containerName + ":latest")

		kubeClient.Namespace("default").LabelNamespace(ctx, map[string]string{
			"secrets-injection": "enabled",
		})
	})

	Context("Use Secrets Injector with Connect", func() {
		BeforeAll(func() {
			deployConnect(ctx)
			deployInjector(ctx)
			deployClientApp(ctx, "client-connect.yaml")
		})

		AfterAll(func() {
			_, err := system.Run("helm", "uninstall", "onepassword-connect")
			Expect(err).NotTo(HaveOccurred())
			kubeClient.Secret("onepassword-connect-token").Delete(ctx)
		})

		runCommonTestCases(ctx)
	})

	Context("Use Secrets Injector with Service Account", func() {
		BeforeAll(func() {
			kubeClient.Secret("onepassword-service-account-token").CreateFromEnvVar(ctx, "OP_SERVICE_ACCOUNT_TOKEN")
			kubeClient.Secret("onepassword-service-account-token").CheckIfExists(ctx)
			deployClientApp(ctx, "client-service-account.yaml")
		})

		runCommonTestCases(ctx)
	})
})

// runCommonTestCases contains test cases that are common to both Connect and Service Account authentication methods.
func runCommonTestCases(ctx context.Context) {
	It("Should inject secret into app pod", func() {
		kubeClient.Pod(map[string]string{
			"app": containerName,
		}).VerifyWebhookInjection(ctx)

		kubeClient.Pod(map[string]string{
			"app": containerName,
		}).VerifySecretsInjected(ctx)
	})
}

func deployConnect(ctx context.Context) {
	root, err := system.GetProjectRoot()
	Expect(err).NotTo(HaveOccurred())

	kubeClient.Secret("onepassword-connect-token").CreateFromEnvVar(ctx, "OP_CONNECT_TOKEN")
	kubeClient.Secret("onepassword-connect-token").CheckIfExists(ctx)

	By("Adding 1Password Connect Helm repository")
	_, err = system.Run("helm", "repo", "add", "1password", "https://1password.github.io/connect-helm-charts/")
	Expect(err).NotTo(HaveOccurred())

	By("Installing 1Password Connect")
	_, err = system.Run("helm", "install", "onepassword-connect", "1password/connect",
		"--namespace", "default",
		"--set-file", "connect.credentials="+root+"/1password-credentials.json",
		"--set", "connect.token=onepassword-connect-token",
		"--wait", "--timeout=5m")
	Expect(err).NotTo(HaveOccurred())

	kubeClient.Deployment("onepassword-connect").WaitDeploymentRolledOut(ctx)
}

func deployInjector(ctx context.Context) {
	_, err := system.Run("make", "deploy")
	Expect(err).NotTo(HaveOccurred())

	kubeClient.Deployment("secrets-injector").WaitDeploymentRolledOut(ctx)
	kubeClient.Webhook("secrets-injector-webhook-config").WaitForWebhookToBeRegistered(ctx)
}

func deployClientApp(ctx context.Context, manifestFileName string) {
	By("Deploy test app")
	kubeClient.Apply(ctx, manifestFileName)
	kubeClient.Deployment(containerName).WaitDeploymentRolledOut(ctx)
}
