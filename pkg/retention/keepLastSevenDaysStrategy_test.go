package retention

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

func TestKeepLastSevenDaysBackupStrategy_emptyInput(t *testing.T) {
	var backups []v1.Backup
	testClock := newMockTimeProvider(t)
	testClock.EXPECT().Now().Return(time.Now())

	strategy := &keepLastSevenDaysStrategy{testClock}

	removed, retained := strategy.FilterForRemoval(backups)

	assert.Empty(t, removed)
	assert.Empty(t, retained)
}

func TestKeepLastSevenDaysBackupStrategy_keepBackupsWithinSevenDays(t *testing.T) {
	now := createDaylightSavingsIrrelevantDate()
	testClock := newMockTimeProvider(t)
	testClock.EXPECT().Now().Return(now)
	currentTime := now
	lastHour := now.Add(-time.Hour)
	yesterday := now.Add(-24 * time.Hour)
	threeDaysBefore := now.AddDate(0, 0, -3)
	sevenDaysBefore := now.AddDate(0, 0, -7)
	backup1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000231"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(currentTime)}}
	backup2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000322"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(lastHour)}}
	backup3 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000353"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(yesterday)}}
	backup4 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000274"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(threeDaysBefore)}}
	backup5 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000675"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(sevenDaysBefore)}}
	backups := []v1.Backup{backup1, backup2, backup3, backup4, backup5}

	strategy := &keepLastSevenDaysStrategy{testClock}

	removed, retained := strategy.FilterForRemoval(backups)

	assert.Empty(t, removed)
	expectedRetained := RetainedBackups{backup1, backup2, backup3, backup4, backup5}
	assert.Equal(t, expectedRetained, retained)
}

func TestKeepLastSevenDaysBackupStrategy_removeBackupsOlderThanSevenDays(t *testing.T) {
	now := createDaylightSavingsIrrelevantDate()
	testClock := newMockTimeProvider(t)
	testClock.EXPECT().Now().Return(now)
	currentTime := now
	lastHour := now.Add(-time.Hour)
	yesterday := now.Add(-24 * time.Hour)
	threeDaysBefore := now.AddDate(0, 0, -3)
	sevenDaysBefore := now.AddDate(0, 0, -7)
	eightDaysBefore := now.AddDate(0, 0, -8)
	nineDaysBefore := now.AddDate(0, 0, -9)
	oneMonthBefore := now.AddDate(0, -1, 0)
	oneYearBefore := now.AddDate(-1, 0, 0)
	backup1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000234"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(currentTime)}}
	backup2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000321"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(lastHour)}}
	backup3 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000356"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(yesterday)}}
	backup4 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000274"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(threeDaysBefore)}}
	backup5 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000675"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(sevenDaysBefore)}} // Perfect hit: Should not be after the same time
	remove1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888343"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(eightDaysBefore)}}
	remove2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888421"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(nineDaysBefore)}}
	remove3 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888483"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(oneMonthBefore)}}
	remove4 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888847"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(oneYearBefore)}}
	backups := []v1.Backup{backup1, backup2, backup3, backup4, backup5, remove1, remove2, remove3, remove4}

	strategy := &keepLastSevenDaysStrategy{testClock}

	removed, retained := strategy.FilterForRemoval(backups)

	expectedRemoved := RemovedBackups{remove1, remove2, remove3, remove4}
	assert.Equal(t, expectedRemoved, removed)
	expectedRetained := RetainedBackups{backup1, backup2, backup3, backup4, backup5}
	assert.Equal(t, expectedRetained, retained)
}

func createDaylightSavingsIrrelevantDate() time.Time {
	return time.Date(2019, 01, 01, 1, 1, 0, 0, time.UTC)
}

func TestKeepLastSevenDaysBackupStrategy_removeBackupsOlderThanSevenDays_reversedOrder(t *testing.T) {
	now := createDaylightSavingsIrrelevantDate()
	testClock := newMockTimeProvider(t)
	testClock.EXPECT().Now().Return(now)
	currentTime := now
	lastHour := now.Add(-time.Hour)
	yesterday := now.Add(-24 * time.Hour)
	threeDaysBefore := now.AddDate(0, 0, -3)
	sevenDaysBefore := now.AddDate(0, 0, -7)
	eightDaysBefore := now.AddDate(0, 0, -8)
	nineDaysBefore := now.AddDate(0, 0, -9)
	oneMonthBefore := now.AddDate(0, -1, 0)
	oneYearBefore := now.AddDate(-1, 0, 0)
	backup1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000231"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(currentTime)}}
	backup2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000322"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(lastHour)}}
	backup3 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000353"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(yesterday)}}
	backup4 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000274"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(threeDaysBefore)}}
	backup5 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000675"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(sevenDaysBefore)}}
	backup6 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888343"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(eightDaysBefore)}}
	backup7 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888421"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(nineDaysBefore)}}
	backup8 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888483"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(oneMonthBefore)}}
	backup9 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888847"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(oneYearBefore)}}
	backups := []v1.Backup{backup9, backup8, backup7, backup6, backup5, backup4, backup3, backup2, backup1}

	strategy := &keepLastSevenDaysStrategy{testClock}

	removed, retained := strategy.FilterForRemoval(backups)

	expectedRemoved := RemovedBackups{backup9, backup8, backup7, backup6}
	assert.Equal(t, expectedRemoved, removed)
	expectedRetained := RetainedBackups{backup5, backup4, backup3, backup2, backup1}
	assert.Equal(t, expectedRetained, retained)
}

func TestLastSevenDays_daylightSavings(t *testing.T) {
	// current 2018-04-03T08:23:51+02:00 is before seven days 2019-03-27T08:22:51+01:00
	daylightSavingsSwitchTime := time.Date(2019, 04, 03, 12, 00, 00, 00, time.Local)
	now := daylightSavingsSwitchTime
	testClock := newMockTimeProvider(t)
	testClock.EXPECT().Now().Return(now)

	currentTime := now
	lastHour := now.Add(-time.Hour)
	yesterday := now.Add(-24 * time.Hour)
	threeDaysBefore := now.AddDate(0, 0, -3)
	sixDaysBefore := now.AddDate(0, 0, -6)
	sevenDaysBefore := now.AddDate(0, 0, -7)
	sevenDays2HoursBefore := now.AddDate(0, 0, -7).Add(1 * time.Hour)
	sevenDaysMinus1HourBefore := now.AddDate(0, 0, -7).Add(-1 * time.Hour)
	eightDaysBefore := now.AddDate(0, 0, -8)
	nineDaysBefore := now.AddDate(0, 0, -9)
	oneMonthBefore := now.AddDate(0, -1, 0)
	oneYearBefore := now.AddDate(-1, 0, 0)
	keep1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000121"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(currentTime)}}
	keep2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000322"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(lastHour)}}
	keep3 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000453"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(yesterday)}}
	keep4 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000654"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(threeDaysBefore)}}
	keep5 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000785"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(sixDaysBefore)}}
	keep6Plus := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "cafebabe"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(sevenDays2HoursBefore)}}   // seven days, one hour and one daylight savings before
	keep7 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000951"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(sevenDaysBefore)}}             // perfect hit: six days, 23 hours and one hour daylight savings
	remove1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "deadbeef"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(sevenDaysMinus1HourBefore)}} // 6 days, 23 hours due to daylight savings
	remove2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888341"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(eightDaysBefore)}}
	remove3 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888422"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(nineDaysBefore)}}
	remove4 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888483"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(oneMonthBefore)}}
	remove5 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888844"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(oneYearBefore)}}
	backups := []v1.Backup{remove5, remove4, remove3, remove2, remove1, keep7, keep6Plus, keep5, keep4, keep3, keep2, keep1}

	strategy := &keepLastSevenDaysStrategy{testClock}

	removed, retained := strategy.FilterForRemoval(backups)

	expectedRemoved := RemovedBackups{remove5, remove4, remove3, remove2, remove1}
	assert.Equal(t, expectedRemoved, removed)
	expectedRetained := RetainedBackups{keep7, keep6Plus, keep5, keep4, keep3, keep2, keep1}
	assert.Equal(t, expectedRetained, retained)
}

func TestKeepLastSevenDaysBackupStrategy_removeBackupsOlderThanSevenDays_multipleBackupsPerDay(t *testing.T) {
	now := time.Now()
	testClock := newMockTimeProvider(t)
	testClock.EXPECT().Now().Return(now)

	currentTime := now
	lastHour := now.Add(-time.Hour)
	yesterday := now.Add(-24 * time.Hour)
	yesterday2 := now.Add(-26 * time.Hour)
	yesterday6 := now.Add(-30 * time.Hour)
	threeDaysBefore := now.AddDate(0, 0, -3)
	eightDaysBefore := now.AddDate(0, 0, -8)
	nineDaysBefore := now.AddDate(0, 0, -9)
	oneMonthBefore := now.AddDate(0, -1, 0)
	oneYearBefore := now.AddDate(-1, 0, 0)
	keep1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000231"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(currentTime)}}
	keep2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000322"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(lastHour)}}
	keep3 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000353"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(yesterday)}}
	keep4 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000274"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(yesterday2)}}
	keep5 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000675"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(yesterday6)}}
	keep6 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000376"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(threeDaysBefore)}}
	remove1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888741"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(eightDaysBefore)}}
	remove2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888262"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(nineDaysBefore)}}
	remove3 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888533"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(oneMonthBefore)}}
	remove4 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888544"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(oneYearBefore)}}
	backups := []v1.Backup{keep1, keep2, keep3, keep4, keep5, keep6, remove1, remove2, remove3, remove4}

	strategy := &keepLastSevenDaysStrategy{testClock}

	removed, retained := strategy.FilterForRemoval(backups)

	expectedRemoved := RemovedBackups{remove1, remove2, remove3, remove4}
	assert.Equal(t, expectedRemoved, removed)
	expectedRetained := RetainedBackups{keep1, keep2, keep3, keep4, keep5, keep6}
	assert.Equal(t, expectedRetained, retained)
}

func TestKeepLastSevenDaysBackupStrategy_timeFormatWithNanoseconds(t *testing.T) {
	now := time.Now()
	testClock := newMockTimeProvider(t)
	testClock.EXPECT().Now().Return(now)

	currentTime := now.Add(1 * time.Second)
	eightDaysBefore := now.AddDate(0, 0, -8)
	backup1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "00000123"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(currentTime)}}
	backup2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "88888321"}, Status: v1.BackupStatus{StartTimestamp: metav1.NewTime(eightDaysBefore)}}
	backups := []v1.Backup{backup1, backup2}

	strategy := &keepLastSevenDaysStrategy{testClock}

	removed, retained := strategy.FilterForRemoval(backups)

	assert.Equal(t, RemovedBackups{backup2}, removed)
	assert.Equal(t, RetainedBackups{backup1}, retained)
}

func Test_keepLastSevenDaysStrategy_GetName(t *testing.T) {
	// given
	sut := &keepLastSevenDaysStrategy{}

	// when
	name := sut.GetName()

	// then
	assert.Equal(t, StrategyId("keepLastSevenDays"), name)
}

func Test_newKeepLastSevenDaysStrategy(t *testing.T) {
	strategy := newKeepLastSevenDaysStrategy()
	assert.NotEmpty(t, strategy)
}
