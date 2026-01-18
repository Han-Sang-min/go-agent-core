package agent

import (
	"context"
	"os"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesEnv struct {
	client kubernetes.Interface

	podName   string
	namespace string
	nodeName  string

	mu       sync.Mutex
	cached   KubernetesMeta
	cachedAt time.Time
	ttl      time.Duration
}

func NewKubernetesEnv() (*KubernetesEnv, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &KubernetesEnv{
		client:    cs,
		podName:   os.Getenv("POD_NAME"),
		namespace: os.Getenv("POD_NAMESPACE"),
		nodeName:  os.Getenv("NODE_NAME"),
		ttl:       2 * time.Minute,
	}, nil
}

func (e *KubernetesEnv) K8sMeta(ctx context.Context) (KubernetesMeta, error) {
	e.mu.Lock()
	if e.cached.Valid && time.Since(e.cachedAt) < e.ttl {
		out := e.cached
		e.mu.Unlock()
		return out, nil
	}
	e.mu.Unlock()

	meta := KubernetesMeta{
		Namespace: e.namespace,
		PodName:   e.podName,
		NodeName:  e.nodeName,
		Valid:     false,
	}

	if e.namespace == "" || e.podName == "" {
		return meta, nil
	}

	pod, err := e.client.CoreV1().Pods(e.namespace).Get(ctx, e.podName, metav1.GetOptions{})
	if err == nil {
		meta.PodUID = string(pod.UID)
		meta.PodIP = pod.Status.PodIP
		if meta.NodeName == "" {
			meta.NodeName = pod.Spec.NodeName
		}
	}

	if meta.NodeName != "" {
		node, nerr := e.client.CoreV1().Nodes().Get(ctx, meta.NodeName, metav1.GetOptions{})
		if nerr == nil {
			meta.NodeUID = string(node.UID)
			meta.NodeLabels = node.Labels
		}
	}

	if meta.Namespace != "" && meta.PodName != "" {
		meta.Valid = true
	}

	e.mu.Lock()
	e.cached = meta
	e.cachedAt = time.Now()
	e.mu.Unlock()

	return meta, nil
}

func isKubernetes() bool {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token")
	return err == nil
}
