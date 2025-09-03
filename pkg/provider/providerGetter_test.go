package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

var testCtx = context.Background()

const testNamespace = "my-namespace"

func TestGetAll(t *testing.T) {
	t.Run("should only valid and ready providers", func(t *testing.T) {
		// given
		oldKnownProviders := knownProviders
		oldNewVeleroFunc := NewVeleroProvider
		defer func() {
			knownProviders = oldKnownProviders
			NewVeleroProvider = oldNewVeleroFunc
		}()

		providerMock := NewMockProvider(t)
		called := false
		providerMock.EXPECT().CheckReady(testCtx).RunAndReturn(func(ctx context.Context) error {
			if called {
				return nil
			}
			called = true
			return assert.AnError
		}).Twice()

		NewVeleroProvider = func(client K8sClient, ecoSystemClient EcosystemClientSet, recorder EventRecorder, namespace string) Provider {
			return providerMock
		}

		recorderMock := NewMockEventRecorder(t)
		recorderMock.EXPECT().Event(mock.Anything, mock.Anything, mock.Anything, mock.Anything)

		knownProviders = []v1.Provider{"invalid", "velero", "", "k10"}

		// when
		providers := GetAll(testCtx, testNamespace, recorderMock, nil, nil)

		// then
		assert.Len(t, providers, 1)
	})
}
