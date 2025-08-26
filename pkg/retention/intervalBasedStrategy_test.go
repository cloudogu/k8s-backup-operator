package retention

import (
	"testing"
	"time"

	v1 "github.com/cloudogu/k8s-backup-lib/pkg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

// Sample Calendar
var calendarRetentionStrategyKeep7Days1Month1Quarter1Year = newIntervalCalendar(KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy).addTimeIntervals(
	[]timeInterval{
		newTimeInterval("sevenDays", 0, 7, keepAllIntervalMode),
		newTimeInterval("thirtyDays", 8, 30, keepOldestIntervalMode),
		newTimeInterval("ninetyDays", 31, 90, keepOldestIntervalMode),
		newTimeInterval("oneHundredEightyDays", 91, 180, keepOldestIntervalMode),
		newTimeInterval("threeHundredSixtyDays", 181, 360, keepOldestIntervalMode),
	})

func TestKeepLastSevenDaysBackupExtendedStrategy(t *testing.T) {
	now := time.Date(2019, 03, 01, 1, 1, 0, 0, time.UTC)
	testClock := newMockTimeProvider(t)
	testClock.EXPECT().Now().Return(now)
	// 22.02.2019 (7 days before 01.03.2019)
	sevenDaysBefore := now.AddDate(0, 0, -7)
	eightDaysBefore := now.AddDate(0, 0, -8)
	oneMonthBefore := now.AddDate(0, 0, -30)
	days89Before := now.AddDate(0, 0, -89)
	oneQuarterBefore := now.AddDate(0, 0, -90)
	days95Before := now.AddDate(0, 0, -95)
	days179Before := now.AddDate(0, 0, -179)
	days360Before := now.AddDate(0, 0, -360)
	days361Before := now.AddDate(0, 0, -361)
	backupToday := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000001"},
		Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(now)}}
	backup7Days := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000002"},
		Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(sevenDaysBefore)}}
	backup8Days := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000003"},
		Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(eightDaysBefore)}}
	backup30Days := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000004"},
		Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(oneMonthBefore)}}
	backup89Days := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000005"},
		Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(days89Before)}}
	backup90Days := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000006"},
		Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(oneQuarterBefore)}}
	backup95Days := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000007"},
		Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(days95Before)}}
	backup179Days := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000008"},
		Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(days179Before)}}
	backup360Days := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000009"},
		Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(days360Before)}}
	backup361Days := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000010"},
		Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(days361Before)}}
	backups := []v1.Backup{backupToday, backup7Days, backup8Days, backup30Days, backup90Days, backup89Days, backup95Days,
		backup179Days, backup360Days, backup361Days}

	strategy := newIntervalBasedStrategy(KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy, calendarRetentionStrategyKeep7Days1Month1Quarter1Year, testClock)

	removed, retained := strategy.FilterForRemoval(backups)

	expectedRemoved := RemovedBackups{backup361Days, backup95Days, backup89Days, backup8Days}
	assert.ElementsMatch(t, expectedRemoved, removed)
	expectedRetained := RetainedBackups{backupToday, backup7Days, backup30Days, backup90Days, backup179Days, backup360Days}
	assert.ElementsMatch(t, expectedRetained, retained)
}

func TestKeepLastSevenDaysBackupExtendedStrategyWithEmptyBackups(t *testing.T) {
	testClock := newMockTimeProvider(t)
	var backups []v1.Backup

	strategy := newIntervalBasedStrategy(KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy, calendarRetentionStrategyKeep7Days1Month1Quarter1Year, testClock)

	removed, retained := strategy.FilterForRemoval(backups)

	assert.Empty(t, removed)
	assert.Empty(t, retained)
}

func Test_retainOldestBackup(t *testing.T) {
	t.Run("should remove and retain nothing for empty list", func(t *testing.T) {
		// when
		remove, retain := retainOldestBackup(nil)

		// then
		assert.Equal(t, RemovedBackups{}, remove)
		assert.Equal(t, RetainedBackups{}, retain)
	})
}
