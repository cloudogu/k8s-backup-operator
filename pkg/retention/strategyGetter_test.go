package retention

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewStrategyGetter(t *testing.T) {
	getter := NewStrategyGetter()
	assert.NotNil(t, getter)
}

func TestStrategyGetter_Get(t *testing.T) {
	tests := []struct {
		name         string
		strategyName StrategyId
		want         Strategy
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name:         "should return keepAll strategy",
			strategyName: StrategyId("keepAll"),
			want:         &keepAllStrategy{},
			wantErr:      assert.NoError,
		},
		{
			name:         "should return removeAllButKeepLatest strategy",
			strategyName: StrategyId("removeAllButKeepLatest"),
			want:         &removeAllButKeepLatestStrategy{},
			wantErr:      assert.NoError,
		},
		{
			name:         "should return keepLastSevenDays strategy",
			strategyName: StrategyId("keepLastSevenDays"),
			want:         &keepLastSevenDaysStrategy{clock: &clock{}},
			wantErr:      assert.NoError,
		},
		{
			name:         "should return keep7Days1Month1Quarter1Year strategy",
			strategyName: StrategyId("keep7Days1Month1Quarter1Year"),
			want: &intervalBasedStrategy{
				name: StrategyId("keep7Days1Month1Quarter1Year"),
				intervalCalendar: intervalCalendar{
					name: StrategyId("keep7Days1Month1Quarter1Year"),
					timeIntervals: []timeInterval{
						newTimeInterval("sevenDays", 0, 7, "ALL"),
						newTimeInterval("thirtyDays", 8, 30, "OLDEST"),
						newTimeInterval("ninetyDays", 31, 90, "OLDEST"),
						newTimeInterval("oneHundredEightyDays", 91, 180, "OLDEST"),
						newTimeInterval("threeHundredSixtyDays", 181, 360, "OLDEST"),
					},
				},
				clock: &clock{},
			},
			wantErr: assert.NoError,
		},
		{
			name:         "should fail on invalid strategy",
			strategyName: StrategyId("invalid"),
			want:         nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "no matching strategy for name \"invalid\"")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sg := &StrategyGetter{}
			got, err := sg.Get(tt.strategyName)
			if !tt.wantErr(t, err, fmt.Sprintf("Get(%v)", tt.strategyName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Get(%v)", tt.strategyName)
		})
	}
}
