package specs

import (
	"flag"
	"path/filepath"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var k8sClientSet kubernetes.Interface
var k8sClient client.Client

func TestSpecs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Specs Suite")
}

var _ = BeforeSuite(func() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "k3ces.localdomain"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	Expect(err).ShouldNot(HaveOccurred())

	k8sClientSet, err = kubernetes.NewForConfig(config)
	Expect(err).ShouldNot(HaveOccurred())

	k8sClient, err = client.New(config, client.Options{})
	Expect(err).ShouldNot(HaveOccurred())

	utilruntime.Must(velerov1.AddToScheme(k8sClient.Scheme()))
	utilruntime.Must(backupv1.AddToScheme(k8sClient.Scheme()))

})
