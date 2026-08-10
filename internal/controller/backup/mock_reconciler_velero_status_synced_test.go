package backup

import (
	"context"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
)

func (m *mockReconciler) ensureVeleroStatusSynced(ctx context.Context, backup *v1.Backup, logger logr.Logger) (action, error) {
	ret := m.Called(ctx, backup, logger)
	if len(ret) == 0 {
		panic("no return value specified for ensureVeleroStatusSynced")
	}
	return ret.Get(0).(action), ret.Error(1)
}

type mockReconcilerEnsureVeleroStatusSyncedCall struct {
	*mock.Call
}

func (e *mockReconciler_Expecter) ensureVeleroStatusSynced(ctx, backup, logger interface{}) *mockReconcilerEnsureVeleroStatusSyncedCall {
	return &mockReconcilerEnsureVeleroStatusSyncedCall{Call: e.mock.On("ensureVeleroStatusSynced", ctx, backup, logger)}
}

func (c *mockReconcilerEnsureVeleroStatusSyncedCall) Return(nextAction action, err error) *mockReconcilerEnsureVeleroStatusSyncedCall {
	c.Call.Return(nextAction, err)
	return c
}
