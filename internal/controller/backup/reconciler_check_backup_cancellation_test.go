package backup

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcilerCheckBackupCancellation(t *testing.T) {
	t.Run("If the time window has not yet expired, set condition and proceed to the next step", func(t *testing.T) {
		baseTime := time.Now()
		backup := newBackupForControllerTest("ns", "backup")
		backup.CreationTimestamp = metav1.NewTime(baseTime)
		backupConfigMap := newBackupConfigMapForReconcilerTest(10)
		var configMapGetCallCount = 0
		var statusPatchCallCount = 0
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, backupConfigMap).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if reflect.TypeOf(obj) == reflect.TypeFor[*corev1.ConfigMap]() {
						configMapGetCallCount++
					}
					return client.Get(ctx, key, obj, opts...)
				},
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					statusPatchCallCount++
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		clockMock := NewMockClock(t)
		clockMock.EXPECT().
			Now().
			Return(baseTime.Add(10*time.Minute - time.Millisecond))
		reconciler := NewReconciler(fakeClient, nil, clockMock)

		nextAction, err := reconciler.checkBackupCancellation(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		canceledCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCanceled)
		assert.NotNil(t, canceledCondition)
		assert.Equal(t, metav1.ConditionFalse, canceledCondition.Status)
		assert.Equal(t, reasonTimeWindowNotExpired, canceledCondition.Reason)

		assert.Equal(t, 1, configMapGetCallCount)
		assert.Equal(t, 1, statusPatchCallCount)
	})

	t.Run("If the time window has expired and the backup has not started, set canceled to true and abort", func(t *testing.T) {
		baseTime := time.Now()
		backup := newBackupForControllerTest("ns", "backup")
		backup.CreationTimestamp = metav1.NewTime(baseTime)
		backupConfigMap := newBackupConfigMapForReconcilerTest(10)
		var configMapGetCallCount = 0
		var statusPatchCallCount = 0
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, backupConfigMap).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if reflect.TypeOf(obj) == reflect.TypeFor[*corev1.ConfigMap]() {
						configMapGetCallCount++
					}
					return client.Get(ctx, key, obj, opts...)
				},
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					statusPatchCallCount++
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		clockMock := NewMockClock(t)
		clockMock.EXPECT().
			Now().
			Return(baseTime.Add(10*time.Minute + time.Millisecond))
		reconciler := NewReconciler(fakeClient, nil, clockMock)

		require.True(t, backup.Status.StartTimestamp.IsZero())
		require.True(t, backup.Status.CompletionTimestamp.IsZero())

		nextAction, err := reconciler.checkBackupCancellation(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Abort, nextAction)

		canceledCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCanceled)
		assert.NotNil(t, canceledCondition)
		assert.Equal(t, metav1.ConditionTrue, canceledCondition.Status)
		assert.Equal(t, reasonTimeWindowExpired, canceledCondition.Reason)
		assert.Equal(t, messageTimeWindowExpiredBackupNotStarted, canceledCondition.Message)

		assert.Equal(t, 1, configMapGetCallCount)
		assert.Equal(t, 1, statusPatchCallCount)
	})
}

func newBackupConfigMapForReconcilerTest(retryLimitInMinutes int) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      backupConfigMapName,
		},
		Data: map[string]string{
			backupRetryTimeLimitKey: strconv.Itoa(retryLimitInMinutes),
		},
	}
}
