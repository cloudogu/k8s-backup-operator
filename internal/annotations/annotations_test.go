package annotations

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetBackupAnnotations(t *testing.T) {
	tests := []struct {
		name       string
		input      metav1.ObjectMeta
		wantResult map[string]string
	}{
		{
			name: "EmptyAnnotations",
			input: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
			wantResult: map[string]string{},
		},
		{
			name: "NoRelevantAnnotations",
			input: metav1.ObjectMeta{
				Annotations: map[string]string{
					"irrelevant.annotation": "value1",
					"another.irrelevant":    "value2",
				},
			},
			wantResult: map[string]string{},
		},
		{
			name: "SingleRelevantAnnotation",
			input: metav1.ObjectMeta{
				Annotations: map[string]string{
					BlueprintIdAnnotation:   "blueprint-123",
					"irrelevant.annotation": "value1",
				},
			},
			wantResult: map[string]string{
				BlueprintIdAnnotation: "blueprint-123",
			},
		},
		{
			name: "MultipleRelevantAnnotations",
			input: metav1.ObjectMeta{
				Annotations: map[string]string{
					BlueprintIdAnnotation:   "blueprint-123",
					DogusAnnotation:         "dogus-abc",
					"irrelevant.annotation": "value1",
				},
			},
			wantResult: map[string]string{
				BlueprintIdAnnotation: "blueprint-123",
				DogusAnnotation:       "dogus-abc",
			},
		},
		{
			name: "OnlyRelevantAnnotations",
			input: metav1.ObjectMeta{
				Annotations: map[string]string{
					BlueprintIdAnnotation: "blueprint-123",
					DogusAnnotation:       "dogus-abc",
				},
			},
			wantResult: map[string]string{
				BlueprintIdAnnotation: "blueprint-123",
				DogusAnnotation:       "dogus-abc",
			},
		},
		{
			name: "NilAnnotations",
			input: metav1.ObjectMeta{
				Annotations: nil,
			},
			wantResult: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBackupAnnotations(tt.input)
			if len(got) != len(tt.wantResult) {
				t.Errorf("expected %d annotations, got %d", len(tt.wantResult), len(got))
			}
			for key, val := range tt.wantResult {
				if got[key] != val {
					t.Errorf("expected annotation %s to have value %s, got %s", key, val, got[key])
				}
			}
		})
	}
}
