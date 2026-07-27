package restore

import (
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestRepeatedReconciliationOfACompletedRestorePerformsNoWritesOrExternalActions(t *testing.T) {
	restore := &backupv1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: testRestore, Namespace: testNamespace},
		Status:     backupv1.RestoreStatus{Status: backupv1.RestoreStatusCompleted},
	}

	restoreClient := newMockEcosystemRestoreInterface(t)
	restoreClient.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(restore, nil).Twice()
	v1Client := newMockEcosystemV1Alpha1Interface(t)
	v1Client.EXPECT().Restores(testNamespace).Return(restoreClient).Twice()
	clientSet := newMockEcosystemInterface(t)
	clientSet.EXPECT().EcosystemV1Alpha1().Return(v1Client).Twice()

	reconciler := &restoreReconciler{
		namespace: testNamespace,
		clientSet: clientSet,
	}

	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testRestore}}

	for range 2 {
		result, err := reconciler.Reconcile(testCtx, request)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	}
}

func TestAParentNotFoundRacePerformsNoWritesOrExternalActions(t *testing.T) {
	notFound := apierrors.NewNotFound(
		schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "restores"},
		testRestore,
	)

	restoreClient := newMockEcosystemRestoreInterface(t)
	restoreClient.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(nil, notFound).Twice()
	v1Client := newMockEcosystemV1Alpha1Interface(t)
	v1Client.EXPECT().Restores(testNamespace).Return(restoreClient).Twice()
	clientSet := newMockEcosystemInterface(t)
	clientSet.EXPECT().EcosystemV1Alpha1().Return(v1Client).Twice()

	reconciler := &restoreReconciler{
		namespace: testNamespace,
		clientSet: clientSet,
	}
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testRestore}}

	for range 2 {
		result, err := reconciler.Reconcile(testCtx, request)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	}
}
