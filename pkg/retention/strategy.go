package retention

type StrategyId string

const (
	// KeepAllStrategy retention policy:
	// The following table gives an overview of the behavior within this strategy:
	//
	// | retained backups | time period |
	// |------------------|-------------|
	// | ALL              | ∞           |
	//
	// The maximum of saved backups is ∞.
	KeepAllStrategy StrategyId = "keepAll"

	// RemoveAllButKeepLatestStrategy retention policy:
	// The following table gives an overview of the behavior within this strategy:
	//
	// | retained backups | time period |
	// |------------------|-------------|
	// | 1                | ∞           |
	//
	// The maximum of saved backups is 1.
	RemoveAllButKeepLatestStrategy StrategyId = "removeAllButKeepLatest"

	// KeepLastSevenDaysStrategy retention policy:
	// The following table gives an overview of the behavior within this strategy:
	//
	// | retained backups | time period |
	// |------------------|-------------|
	// | ALL              | 1-7 days    |
	//
	// The maximum of saved backups is 7 (without consideration of manual backups).
	KeepLastSevenDaysStrategy StrategyId = "keepLastSevenDays"

	// KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy retention policy:
	//
	// The following table gives an overview of the behaviour within this strategy:
	//
	// | retained backups |  time period    |
	// | ALL              |  0 - 7 days     |
	// | 1                |  8 - 30 days    |
	// | 1                |  31 - 90 days   |
	// | 1                |  91 - 180 days  |
	// | 1                |  181 - 360 days |
	//
	// The maximum of saved backups is 11 (without consideration of manual backups)
	// Between interval borders the oldest backup is moving.
	KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy StrategyId = "keep7Days1Month1Quarter1Year"
)
