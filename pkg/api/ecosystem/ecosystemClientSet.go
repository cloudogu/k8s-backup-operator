package ecosystem

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type Interface interface {
	kubernetes.Interface
	EcosystemV1Alpha1() V1Alpha1Interface
}

type V1Alpha1Interface interface {
	BackupsGetter
	RestoresGetter
}

type BackupsGetter interface {
	Backups(namespace string) BackupInterface
}

type RestoresGetter interface {
	Restores(namespace string) RestoreInterface
}

// NewClientSet creates a new instance of the client set for this operator.
func NewClientSet(config *rest.Config, clientSet *kubernetes.Clientset) (*ClientSet, error) {
	backupRestoreClient, err := NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &ClientSet{
		Interface:         clientSet,
		ecosystemV1Alpha1: backupRestoreClient,
	}, nil
}

type ClientSet struct {
	kubernetes.Interface
	ecosystemV1Alpha1 V1Alpha1Interface
}

func (cs *ClientSet) EcosystemV1Alpha1() V1Alpha1Interface {
	return cs.ecosystemV1Alpha1
}

// NewForConfig creates a new V1Alpha1Client for a given rest.Config.
func NewForConfig(c *rest.Config) (*V1Alpha1Client, error) {
	config := *c
	gv := schema.GroupVersion{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version}
	config.ContentConfig.GroupVersion = &gv
	config.APIPath = "/apis"

	s := scheme.Scheme
	err := v1.AddToScheme(s)
	if err != nil {
		return nil, err
	}

	metav1.AddToGroupVersion(s, gv)
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &V1Alpha1Client{restClient: client}, nil
}

type V1Alpha1Client struct {
	restClient rest.Interface
}

func (brc *V1Alpha1Client) Backups(namespace string) BackupInterface {
	return &backupClient{
		client: brc.restClient,
		ns:     namespace,
	}
}

func (brc *V1Alpha1Client) Restores(namespace string) RestoreInterface {
	return &restoreClient{
		client: brc.restClient,
		ns:     namespace,
	}
}
