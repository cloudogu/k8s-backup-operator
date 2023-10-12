package restore

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRestoreManager(t *testing.T) {
	// given
	clientMock := newMockEcosystemInterface(t)
	recorderMock := newMockEventRecorder(t)

	// when
	actual := NewRestoreManager(clientMock, recorderMock)

	// then
	assert.NotEmpty(t, actual)
}
