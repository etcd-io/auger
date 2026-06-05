package e2e

import (
	"bytes"
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

	if got.Name != "kubernetes" {
		t.Errorf("Got service name: %s, expected kubernetes", got.Name)
	}

	if got.Namespace != "default" {
		t.Errorf("Got service namespace: %s, expected default", got.Namespace)
	}

	if got.Kind != "Service" {
		t.Errorf("Got service type: %s, expected Service", got.Kind)
	}
}

func TestGetWithLimit(t *testing.T) {
	// serviceaccounts has multiple records across namespaces in a kwok cluster,
	// so it is a reliable target for verifying --limit truncates the result set.
	out, err := exec.Command(augerctl,
		"--endpoints", endpoint,
		"get", "serviceaccounts",
		"--limit", "1",
	).Output()
	if err != nil {
		t.Fatal(err)
	}

	// The YAML printer emits each object as a document separated by `---`.
	// With --limit 1 we expect exactly one object document.
	docs := bytes.Split(bytes.TrimSpace(out), []byte("\n---\n"))
	if len(docs) != 1 {
		t.Errorf("expected 1 document with --limit 1, got %d", len(docs))
	}

	got := corev1.ServiceAccount{}
	if err := yaml.Unmarshal(docs[0], &got); err != nil {
		t.Fatalf("decode first document: %v", err)
	}
	if got.Kind != "ServiceAccount" {
		t.Errorf("Got kind: %s, expected ServiceAccount", got.Kind)
	}
}
