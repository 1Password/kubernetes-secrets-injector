package webhook

import (
	"os"
	"os/exec"
	"testing"
)

func TestShouldExitProgramIfNoEnvVarsProvided(t *testing.T) {
	if os.Getenv("SHOULD_EXIT") == "1" {
		CreateWebhookConfig()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestShouldExitProgramIfNoEnvVarsProvided")
	cmd.Env = append(os.Environ(), "SHOULD_EXIT=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("Process ran with err %v, should exit with status 1", err)
}

func TestShouldPopulateServiceAccountConfig(t *testing.T) {
	t.Setenv(serviceAccountSecretNameEnv, "secret-name")
	t.Setenv(serviceAccountSecretKeyEnv, "token")

	config := CreateWebhookConfig()

	if config.ServiceAccount == nil {
		t.Fatal("Service Account config should be populated")
	}
}

func TestShouldPopulateConnectConfig(t *testing.T) {
	t.Setenv(connectTokenSecretNameEnv, "secret-name")
	t.Setenv(connectTokenSecretKeyEnv, "token")
	t.Setenv(connectHostEnv, "http://localhost:8080")

	config := CreateWebhookConfig()

	if config.Connect == nil {
		t.Fatal("Connect config should be populated")
	}
}
