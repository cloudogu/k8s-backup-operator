package retention

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigKeep7Days1Month1Quarter1YearRetentionStrategy(t *testing.T) {
	rs, err := newKeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy()
	assert.Nil(t, err)
	assert.Equal(t, rs.GetName(), KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy)
}
