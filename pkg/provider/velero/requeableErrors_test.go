package velero

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_genericRequeueableError_Error(t *testing.T) {
	sut := &genericRequeueableError{"oh noez", assert.AnError}
	expected := "oh noez: " + assert.AnError.Error()
	assert.Equal(t, expected, sut.Error())
}

func Test_genericRequeueableError_GetRequeueTime(t *testing.T) {
	type args struct {
		requeueTime time.Duration
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		// double the value until the threshold jumps in
		{"1st interval", args{0 * time.Second}, 15 * time.Second},
		{"2nd interval", args{15 * time.Second}, 30 * time.Second},
		{"3rd interval", args{30 * time.Second}, 1 * time.Minute},
		{"11th interval", args{128 * time.Minute}, 256 * time.Minute},
		{"cutoff interval ", args{256 * time.Minute}, 6 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			due := &genericRequeueableError{}
			assert.Equalf(t, tt.want, due.GetRequeueTime(tt.args.requeueTime), "getRequeueTime(%v)", tt.args.requeueTime)
		})
	}
}
