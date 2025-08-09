package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewKubeClient builds a clientset from kubeconfig file if present, otherwise in-cluster config.
func NewKubeClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	if kubeconfigPath == "" {
		if env := os.Getenv("KUBECONFIG"); env != "" {
			kubeconfigPath = env
		} else {
			home, _ := os.UserHomeDir()
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	abs, _ := filepath.Abs(kubeconfigPath)
	if _, err := os.Stat(abs); err == nil {
		cfg, err := clientcmd.BuildConfigFromFlags("", abs)
		if err != nil {
			return nil, fmt.Errorf("build config from kubeconfig: %w", err)
		}
		cs, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("create clientset from kubeconfig: %w", err)
		}
		return cs, nil
	}

	// fallback to in-cluster
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config failed: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create clientset from in-cluster: %w", err)
	}
	return cs, nil
}
