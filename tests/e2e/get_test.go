package e2e

import (
	"os/exec"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestGet(t *testing.T) {
	out, err := exec.Command(augerctl,
		"--endpoints", endpoint,
		"get", "services", "kubernetes", "-n", "default",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	got := corev1.Service{}

	err = yaml.Unmarshal(out, &got)
	if err != nil {
		t.Fatal(err)
	}

	if got.ObjectMeta.Name != "kubernetes" {
		t.Errorf("Got service name: %s, expected kubernetes", got.ObjectMeta.Name)
	}

	if got.ObjectMeta.Namespace != "default" {
		t.Errorf("Got service namespace: %s, expected default", got.ObjectMeta.Namespace)
	}

	if got.TypeMeta.Kind != "Service" {
		t.Errorf("Got service type: %s, expected Service", got.TypeMeta.Kind)
	}
}
