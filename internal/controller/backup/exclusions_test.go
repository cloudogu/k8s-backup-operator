package backup

import (
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	componentv1 "github.com/cloudogu/k8s-component-lib/api/v1"
	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/stretchr/testify/assert"
)

func newDoguForExclusionTest(name string, apiVersion doguv3.DoguApiVersion) doguv3.Dogu {
	return doguv3.Dogu{Name: name, Namespace: "ns", Spec: doguv3.DoguSpec{Name: name, DoguApiVersion: apiVersion}}
}

func TestFindInvalidExclusions(t *testing.T) {
	dogus := []doguv3.Dogu{
		newDoguForExclusionTest("ldap", doguv3.DoguApiVersionV3),
		newDoguForExclusionTest("redmine", doguv3.DoguApiVersionV3),
		newDoguForExclusionTest("cas", doguv3.DoguApiVersionV2),
		newDoguForExclusionTest("unknown-api", "v1"),
	}
	components := []componentv1.Component{
		{Name: "k8s-dogu-operator", Namespace: "ns", Spec: componentv1.ComponentSpec{Name: "k8s-dogu-operator"}},
	}

	tests := []struct {
		name string
		spec backupv1.BackupSpec
		want invalidExclusions
	}{
		{
			name: "no exclusions",
			spec: backupv1.BackupSpec{},
			want: invalidExclusions{},
		},
		{
			name: "installed v3 dogus and components are valid",
			spec: backupv1.BackupSpec{ExcludedDogus: []string{"ldap", "redmine"}, ExcludedComponents: []string{"k8s-dogu-operator"}},
			want: invalidExclusions{},
		},
		{
			name: "dogus which are not installed are invalid",
			spec: backupv1.BackupSpec{ExcludedDogus: []string{"scm", "ldap", "jenkins"}},
			want: invalidExclusions{missingDogus: []string{"scm", "jenkins"}},
		},
		{
			name: "dogus with an unknown api version are not installed as v3 dogus",
			spec: backupv1.BackupSpec{ExcludedDogus: []string{"unknown-api"}},
			want: invalidExclusions{missingDogus: []string{"unknown-api"}},
		},
		{
			name: "v2 dogus are invalid",
			spec: backupv1.BackupSpec{ExcludedDogus: []string{"cas"}},
			want: invalidExclusions{v2Dogus: []string{"cas"}},
		},
		{
			name: "components which are not installed are invalid",
			spec: backupv1.BackupSpec{ExcludedComponents: []string{"k8s-dogu-operator", "k8s-longhorn", "k8s-cert-manager"}},
			want: invalidExclusions{missingComponents: []string{"k8s-longhorn", "k8s-cert-manager"}},
		},
		{
			name: "all kinds of invalid exclusions are collected",
			spec: backupv1.BackupSpec{ExcludedDogus: []string{"cas", "scm"}, ExcludedComponents: []string{"k8s-longhorn"}},
			want: invalidExclusions{missingDogus: []string{"scm"}, v2Dogus: []string{"cas"}, missingComponents: []string{"k8s-longhorn"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, findInvalidExclusions(tt.spec, dogus, components))
		})
	}
}

func TestInvalidExclusions_isEmpty(t *testing.T) {
	assert.True(t, invalidExclusions{}.isEmpty())
	assert.False(t, invalidExclusions{missingDogus: []string{"scm"}}.isEmpty())
	assert.False(t, invalidExclusions{v2Dogus: []string{"cas"}}.isEmpty())
	assert.False(t, invalidExclusions{missingComponents: []string{"k8s-longhorn"}}.isEmpty())
}

func TestInvalidExclusions_message(t *testing.T) {
	t.Run("contains all invalid exclusions", func(t *testing.T) {
		invalid := invalidExclusions{missingDogus: []string{"scm"}, v2Dogus: []string{"cas"}, missingComponents: []string{"k8s-longhorn"}}

		assert.Equal(t, "The excluded dogus [scm] are not installed as v3 dogus. "+
			"The excluded dogus [cas] are v2 dogus, which can only be excluded as a whole with options.ignoreV2Dogus. "+
			"The excluded components [k8s-longhorn] are not installed. "+
			"Create a new backup with valid exclusions.", invalid.message())
	})

	t.Run("is empty without invalid exclusions", func(t *testing.T) {
		assert.Empty(t, invalidExclusions{}.message())
	})

	t.Run("only contains the present kinds", func(t *testing.T) {
		invalid := invalidExclusions{missingComponents: []string{"k8s-longhorn"}}

		assert.Equal(t, "The excluded components [k8s-longhorn] are not installed. Create a new backup with valid exclusions.", invalid.message())
	})
}
