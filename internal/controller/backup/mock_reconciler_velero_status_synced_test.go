package backup

import (
	"context"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
)

func (m *mockReconciler) checkVeleroStatusSynced(ctx context.Context, backup *v1.Backup, logger logr.Logger) (action, error) {
	ret := m.Called(ctx, backup, logger)
	if len(ret) == 0 {
		panic("no return value specified for checkVeleroStatusSynced")
	}
	return ret.Get(0).(action), ret.Error(1)
}

type mockReconcilerCheckVeleroStatusSyncedCall struct {
	*mock.Call
}

func (e *mockReconciler_Expecter) checkVeleroStatusSynced(ctx, backup, logger interface{}) *mockReconcilerCheckVeleroStatusSyncedCall {
	return &mockReconcilerCheckVeleroStatusSyncedCall{Call: e.mock.On("checkVeleroStatusSynced", ctx, backup, logger)}
}

func (c *mockReconcilerCheckVeleroStatusSyncedCall) Return(nextAction action, err error) *mockReconcilerCheckVeleroStatusSyncedCall {
	c.Call.Return(nextAction, err)
	return c
}
