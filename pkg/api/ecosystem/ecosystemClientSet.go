package ecosystem

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

// Interface extends the kubernetes.Interface to add functionality for handling the custom resources of this operator.
type Interface interface {
	kubernetes.Interface
	// EcosystemV1Alpha1 returns a getter for the custom resources of this operator.
	EcosystemV1Alpha1() V1Alpha1Interface
}

// V1Alpha1Interface is a getter for the custom resources of this operator.
type V1Alpha1Interface interface {
	BackupsGetter
	RestoresGetter
	BackupSchedulesGetter
}

type BackupsGetter interface {
	// Backups returns a client for backups in the given namespace.
	Backups(namespace string) BackupInterface
}

type RestoresGetter interface {
	// Restores returns a client for restores in the given namespace.
	Restores(namespace string) RestoreInterface
}

type BackupSchedulesGetter interface {
	// BackupSchedules returns a client for backup schedules in the given namespace.
	BackupSchedules(namespace string) BackupScheduleInterface
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

// ClientSet extends the kubernetes.Interface to add functionality for handling the custom resources of this operator.
type ClientSet struct {
	kubernetes.Interface
	ecosystemV1Alpha1 V1Alpha1Interface
}

// EcosystemV1Alpha1 returns a getter for the custom resources of this operator.
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

type restInterface interface {
	rest.Interface
}

// V1Alpha1Client is a getter for the custom resources of this operator.
type V1Alpha1Client struct {
	restClient restInterface
}

// Backups returns a client for backups in the given namespace.
func (brc *V1Alpha1Client) Backups(namespace string) BackupInterface {
	return &backupClient{
		client: brc.restClient,
		ns:     namespace,
	}
}

// Restores returns a client for restores in the given namespace.
func (brc *V1Alpha1Client) Restores(namespace string) RestoreInterface {
	return &restoreClient{
		client: brc.restClient,
		ns:     namespace,
	}
}

// BackupSchedules returns a client for backup schedules in the given namespace.
func (brc *V1Alpha1Client) BackupSchedules(namespace string) BackupScheduleInterface {
	return &backupScheduleClient{
		client: brc.restClient,
		ns:     namespace,
	}
}
