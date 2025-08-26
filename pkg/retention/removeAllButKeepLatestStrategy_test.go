package retention

import (
	v1 "github.com/cloudogu/k8s-backup-lib/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func Test_removeAllButKeepLatestStrategy_GetName(t *testing.T) {
	// given
	sut := &removeAllButKeepLatestStrategy{}

	// when
	name := sut.GetName()

	// then
	assert.Equal(t, StrategyId("removeAllButKeepLatest"), name)
}

func Test_removeAllButKeepLatestStrategy_FilterForRemoval(t *testing.T) {
	mostRecent := metav1.Now()
	earlier := metav1.NewTime(mostRecent.Add(-24 * time.Hour))
	earliest := metav1.NewTime(earlier.Add(-24 * time.Hour))
	type args struct {
		allBackups []v1.Backup
	}
	tests := []struct {
		name         string
		args         args
		wantRemoved  RemovedBackups
		wantRetained RetainedBackups
	}{
		{
			name:         "should return empty lists if backup list is empty",
			args:         args{allBackups: make([]v1.Backup, 0)},
			wantRemoved:  RemovedBackups{},
			wantRetained: RetainedBackups{},
		},
		{
			name: "should retain latest backup and remove all others",
			args: args{allBackups: []v1.Backup{
				{Status: v1.BackupStatus{StartTimestamp: earlier}},
				{Status: v1.BackupStatus{StartTimestamp: mostRecent}},
				{Status: v1.BackupStatus{StartTimestamp: earliest}},
			}},
			wantRemoved: RemovedBackups{
				{Status: v1.BackupStatus{StartTimestamp: earlier}},
				{Status: v1.BackupStatus{StartTimestamp: earliest}},
			},
			wantRetained: RetainedBackups{{Status: v1.BackupStatus{StartTimestamp: mostRecent}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kls := &removeAllButKeepLatestStrategy{}
			got, got1 := kls.FilterForRemoval(tt.args.allBackups)
			assert.Equalf(t, tt.wantRemoved, got, "FilterForRemoval(%v)", tt.args.allBackups)
			assert.Equalf(t, tt.wantRetained, got1, "FilterForRemoval(%v)", tt.args.allBackups)
		})
	}
}
