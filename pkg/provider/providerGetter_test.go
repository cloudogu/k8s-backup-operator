package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
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
		providerMock.EXPECT().CheckReady(testCtx).Once().Return(assert.AnError)
		providerMock.EXPECT().CheckReady(testCtx).Once().Return(nil)
		callCount := 0
		NewVeleroProvider = func(ecosystemClientSet EcosystemClientSet, recorder EventRecorder, namespace string) (Provider, error) {
			callCount += 1

			if callCount <= 2 {
				return providerMock, nil
			}

			return nil, assert.AnError
		}

		recorderMock := NewMockEventRecorder(t)
		recorderMock.EXPECT().Event(mock.Anything, mock.Anything, mock.Anything, mock.Anything)

		knownProviders = []v1.Provider{"invalid", "", "velero", "", "k10"}

		// when
		providers := GetAll(testCtx, testNamespace, recorderMock, nil)

		// then
		assert.Len(t, providers, 1)
	})
}
