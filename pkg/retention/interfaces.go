package retention

import corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

type configMapClient interface {
	corev1.ConfigMapInterface
}
