package retention

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_clock_Now(t *testing.T) {
	sut := &clock{}
	now := sut.Now()
	assert.NotEmpty(t, now)
}
