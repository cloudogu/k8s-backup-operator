package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestEnsureConditionsInitialized(t *testing.T) {
	t.Run("It should seed every workflow condition of a new backup in one write", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		patches := 0
		fakeClient := newFakeClientForEnsureConditionsInitializedTest(t, backup, &patches, nil)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureConditionsInitialized(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Equal(t, 1, patches, "the conditions must be seeded in a single status write")

		stored := &backupv1.Backup{}
		require.NoError(t, fakeClient.Get(context.Background(), backup.GetNamespacedName(), stored))
		assert.Len(t, stored.Status.Conditions, 5)
		for i, initCondition := range initialBackupConditions {
			assert.Equal(t, initCondition.Type, stored.Status.Conditions[i].Type)
			assert.Equal(t, initCondition.Status, stored.Status.Conditions[i].Status)
			assert.Equal(t, initCondition.Reason, stored.Status.Conditions[i].Reason)
			assert.Equal(t, initCondition.Message, stored.Status.Conditions[i].Message)
		}
	})

	t.Run("It should not reset a condition a stage already resolved", func(t *testing.T) {
		backup := withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionPrepared, metav1.ConditionTrue)
		fakeClient := newFakeClientForEnsureConditionsInitializedTest(t, backup, new(int), nil)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureConditionsInitialized(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		stored := &backupv1.Backup{}
		require.NoError(t, fakeClient.Get(context.Background(), backup.GetNamespacedName(), stored))
		prepared := apiMeta.FindStatusCondition(stored.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, prepared)
		assert.Equal(t, metav1.ConditionTrue, prepared.Status)
		assert.Equal(t, "aReason", prepared.Reason, "the resolved condition must keep the reason its stage wrote")
		assert.Len(t, stored.Status.Conditions, len(initialBackupConditions),
			"the remaining conditions must still be seeded")
	})

	t.Run("It should not write at all when every condition is already present", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		for _, condition := range initialBackupConditions {
			apiMeta.SetStatusCondition(&backup.Status.Conditions, condition)
		}
		patches := 0
		fakeClient := newFakeClientForEnsureConditionsInitializedTest(t, backup, &patches, nil)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureConditionsInitialized(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Zero(t, patches, "a backup that carries every condition must not be patched again")
	})

	t.Run("It should abort with a wrapped error when the status write fails", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		patchErr := errors.NewInternalError(assert.AnError)
		fakeClient := newFakeClientForEnsureConditionsInitializedTest(t, backup, new(int), patchErr)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureConditionsInitialized(context.Background(), backup)

		assert.Equal(t, Abort, nextAction)
		require.Error(t, err)
		assert.ErrorIs(t, err, patchErr)
		assert.ErrorContains(t, err, "patch status to initialize the backup conditions")
	})
}

func TestInitialBackupConditionsSeedEveryConditionType(t *testing.T) {
	seeded := map[string]metav1.ConditionStatus{}
	for _, condition := range initialBackupConditions {
		assert.Equal(t, conditions.ReasonPending, condition.Reason, "condition %s", condition.Type)
		assert.NotEmpty(t, condition.Message, "condition %s", condition.Type)
		seeded[condition.Type] = condition.Status
	}

	assert.Equal(t, map[string]metav1.ConditionStatus{
		backupv1.ConditionSucceeded:         metav1.ConditionUnknown,
		backupv1.ConditionPrepared:          metav1.ConditionUnknown,
		backupv1.ConditionProviderSucceeded: metav1.ConditionUnknown,
		backupv1.ConditionCanceled:          metav1.ConditionFalse,
		backupv1.ConditionDeleting:          metav1.ConditionFalse,
	}, seeded)
}

func newFakeClientForEnsureConditionsInitializedTest(
	t *testing.T,
	backup *backupv1.Backup,
	patches *int,
	patchErr error,
) client.WithWatch {
	return newFakeClientBuilder(t).
		WithObjects(backup).
		WithStatusSubresource(backup).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourcePatch: func(
				ctx context.Context,
				c client.Client,
				subResourceName string,
				obj client.Object,
				patch client.Patch,
				opts ...client.SubResourcePatchOption,
			) error {
				*patches++
				if patchErr != nil {
					return patchErr
				}
				return c.Status().Patch(ctx, obj, patch, opts...)
			},
		}).
		Build()
}
