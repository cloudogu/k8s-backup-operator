//go:build acceptance

// Package specs contains cluster acceptance tests. They require a running
// EcoSystem with the operator and Velero deployed and are excluded from
// ordinary unit runs by the `acceptance` build tag. Run them with
// `make acceptance-test`.
package specs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var k8sClientSet kubernetes.Interface
var k8sClient client.Client

func TestSpecs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Specs Suite")
}

var _ = BeforeSuite(func() {
	kubeConfigEnv := os.Getenv("K8S_TEST_CLUSTER_KUBECONFIG")
	if kubeConfigEnv == "" {
		Skip("K8S_TEST_CLUSTER_KUBECONFIG is not set: skipping cluster acceptance tests")
	}
	expectSingleKubeConfigFile(kubeConfigEnv)

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigEnv)
	Expect(err).ShouldNot(HaveOccurred())

	// These specs are destructive, so record the cluster they actually target.
	// Print ONLY on process 1 using fmt.Printf so it bypasses Ginkgo's pass/fail buffer
	if GinkgoParallelProcess() == 1 {
		fmt.Printf("acceptance specs target cluster %s\n", config.Host)
	}
	k8sClientSet, err = kubernetes.NewForConfig(config)
	Expect(err).ShouldNot(HaveOccurred())

	k8sClient, err = client.New(config, client.Options{})
	Expect(err).ShouldNot(HaveOccurred())

	utilruntime.Must(velerov1.AddToScheme(k8sClient.Scheme()))
	utilruntime.Must(backupv1.AddToScheme(k8sClient.Scheme()))
})

// expectSingleKubeConfigFile fails the suite when K8S_TEST_CLUSTER_KUBECONFIG
// holds a KUBECONFIG-style precedence list rather than one file.
//
// These specs are destructive: they delete every dogu and every backup-scope
// labeled ConfigMap, Secret and PVC in the target namespace. With a precedence
// list the target cluster is whichever file happens to come first, which is far
// too easy to get wrong. Refuse to run instead of guessing.
func expectSingleKubeConfigFile(kubeConfigEnv string) {
	files := filepath.SplitList(kubeConfigEnv)
	if len(files) <= 1 {
		return
	}

	Fail(fmt.Sprintf(
		"K8S_TEST_CLUSTER_KUBECONFIG contains %d kubeconfig files, but these acceptance "+
			"tests are destructive and must not guess which cluster to target.\n"+
			"  value: %s\n"+
			"  files: %s\n"+
			"Set K8S_TEST_CLUSTER_KUBECONFIG to a single kubeconfig file, for example:\n"+
			"  K8S_TEST_CLUSTER_KUBECONFIG=%s make run-specs",
		len(files), kubeConfigEnv, strings.Join(files, ", "), files[0]))
}
