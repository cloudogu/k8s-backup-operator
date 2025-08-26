package retention

import (
	v1 "github.com/cloudogu/k8s-backup-lib/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_keepAllStrategy_FilterForRemoval(t *testing.T) {
	t.Run("should keep all and remove none", func(t *testing.T) {
		// given
		sut := &keepAllStrategy{}
		backups := []v1.Backup{{}, {}}

		// when
		remove, retain := sut.FilterForRemoval(backups)

		// then
		assert.Equal(t, RemovedBackups{}, remove)
		assert.Equal(t, RetainedBackups(backups), retain)
		assert.Len(t, retain, 2)
	})
	t.Run("should keep none and remove none on empty backup list", func(t *testing.T) {
		// given
		sut := &keepAllStrategy{}
		var backups []v1.Backup

		// when
		remove, retain := sut.FilterForRemoval(backups)

		// then
		assert.Equal(t, RemovedBackups{}, remove)
		assert.Equal(t, RetainedBackups(nil), retain)
	})
}

func Test_keepAllStrategy_GetName(t *testing.T) {
	// given
	sut := &keepAllStrategy{}

	// when
	name := sut.GetName()

	// then
	assert.Equal(t, StrategyId("keepAll"), name)
}
