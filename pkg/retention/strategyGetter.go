package retention

import "fmt"

type StrategyGetter struct{}

func NewStrategyGetter() *StrategyGetter {
	return &StrategyGetter{}
}

func (sg *StrategyGetter) Get(name StrategyId) (Strategy, error) {
	switch name {
	case KeepAllStrategy:
		return &keepAllStrategy{}, nil
	case RemoveAllButKeepLatestStrategy:
		return &removeAllButKeepLatestStrategy{}, nil
	case KeepLastSevenDaysStrategy:
		return newKeepLastSevenDaysStrategy(), nil
	case KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy:
		strategy, err := newKeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy()
		if err != nil {
			return nil, fmt.Errorf("error creating strategy %q: %w", KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy, err)
		}

		return strategy, nil
	default:
		return nil, fmt.Errorf("no matching strategy for name %q", name)
	}
}
