package agent

import (
	"testing"
)

func TestK8sLabels_Valid(t *testing.T) {
	meta := KubernetesMeta{
		Namespace:  "default",
		PodName:    "my-pod",
		PodUID:     "uid-123",
		PodIP:      "10.0.0.1",
		NodeName:   "node-1",
		NodeUID:    "node-uid-456",
		NodeLabels: map[string]string{"zone": "us-east-1a"},
		Valid:      true,
	}

	labels := k8sLabels(meta)

	if labels == nil {
		t.Fatal("expected labels, got nil")
	}

	cases := map[string]string{
		"namespace": "default",
		"pod_name":  "my-pod",
		"pod_uid":   "uid-123",
		"pod_ip":    "10.0.0.1",
		"node_name": "node-1",
		"node_uid":  "node-uid-456",
		"node_label/zone": "us-east-1a",
	}

	for k, want := range cases {
		if got := labels[k]; got != want {
			t.Errorf("labels[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestK8sLabels_Invalid(t *testing.T) {
	meta := KubernetesMeta{Valid: false}

	labels := k8sLabels(meta)

	if labels != nil {
		t.Errorf("expected nil for invalid meta, got %v", labels)
	}
}

func TestK8sLabels_NoNodeLabels(t *testing.T) {
	meta := KubernetesMeta{
		Namespace: "kube-system",
		PodName:   "agent-abc",
		Valid:     true,
	}

	labels := k8sLabels(meta)

	if labels == nil {
		t.Fatal("expected labels, got nil")
	}
	if labels["namespace"] != "kube-system" {
		t.Errorf("unexpected namespace: %q", labels["namespace"])
	}
	for k := range labels {
		if len(k) > len("node_label/") && k[:len("node_label/")] == "node_label/" {
			t.Errorf("unexpected node_label key: %q", k)
		}
	}
}
