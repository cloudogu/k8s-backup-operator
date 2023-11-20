package retention

import "fmt"

// StrategyGetter is capable of returning a Strategy identified by its name.
type StrategyGetter struct{}

// NewStrategyGetter creates something capable of returning a Strategy.
func NewStrategyGetter() *StrategyGetter {
	return &StrategyGetter{}
}

// Get returns the Strategy implementation identified by the given name.
func (sg *StrategyGetter) Get(name StrategyId) (Strategy, error) {
	switch name {
	case KeepAllStrategy:
		return &keepAllStrategy{}, nil
	case RemoveAllButKeepLatestStrategy:
		return &removeAllButKeepLatestStrategy{}, nil
	case KeepLastSevenDaysStrategy:
		return newKeepLastSevenDaysStrategy(), nil
	case KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy:
		return newKeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy(), nil
	default:
		return nil, fmt.Errorf("no matching strategy for name %q", name)
	}
}
