package backup

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcilerCheckBackupCompletion(t *testing.T) {
	t.Run("if the backup is not completed proceed to the next step", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)

		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock)

		nextAction, err := reconciler.checkBackupCompletion(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("if the backup is completed proceed abort", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		backup.Status.CompletionTimestamp = metav1.Now()
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)

		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock)

		nextAction, err := reconciler.checkBackupCompletion(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Abort, nextAction)
	})
}
