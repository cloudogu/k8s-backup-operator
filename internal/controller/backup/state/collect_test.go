package state

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	componentv1 "github.com/cloudogu/k8s-component-lib/api/v1"
	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testBackupName = "testBackup"
	testNamespace  = "testNamespace"
)

func TestBackupStateCollector_Collect(t *testing.T) {
	tests := []struct {
		name                     string
		k8sClientBuilderModifier func(*fake.ClientBuilder) *fake.ClientBuilder
		backup                   *backupv1.Backup
		want                     BackupState
		wantErr                  assert.ErrorAssertionFunc
	}{
		{
			name: "fail to list blueprints",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithInterceptorFuncs(interceptor.Funcs{
					List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
						if _, ok := list.(*blueprintv3.BlueprintList); ok {
							return assert.AnError
						}
						return client.List(ctx, list, opts...)
					},
				})
			},
			backup: &backupv1.Backup{Name: testBackupName, Namespace: testNamespace},
			want:   BackupState{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError) &&
					assert.ErrorContains(t, err, "get blueprint id: list blueprints")
			},
		},
		{
			name: "fail to list dogus",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithInterceptorFuncs(interceptor.Funcs{
					List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
						if _, ok := list.(*doguv3.DoguList); ok {
							return assert.AnError
						}
						return client.List(ctx, list, opts...)
					},
				})
			},
			backup: &backupv1.Backup{Name: testBackupName, Namespace: testNamespace},
			want:   BackupState{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError) &&
					assert.ErrorContains(t, err, "get v2 and v3 dogus: list dogus")
			},
		},
		{
			name: "fail to list component config maps",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithInterceptorFuncs(interceptor.Funcs{
					List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
						if unstructuredList, ok := list.(*unstructured.UnstructuredList); ok && unstructuredList.GetKind() == "ConfigMapList" {
							return assert.AnError
						}
						return client.List(ctx, list, opts...)
					},
				})
			},
			backup: &backupv1.Backup{Name: testBackupName, Namespace: testNamespace},
			want:   BackupState{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError) &&
					assert.ErrorContains(t, err, "get components: collect backupable items: list component config maps")
			},
		},
		{
			name: "fail to list component secrets",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithInterceptorFuncs(interceptor.Funcs{
					List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
						if unstructuredList, ok := list.(*unstructured.UnstructuredList); ok && unstructuredList.GetKind() == "SecretList" {
							return assert.AnError
						}
						return client.List(ctx, list, opts...)
					},
				})
			},
			backup: &backupv1.Backup{Name: testBackupName, Namespace: testNamespace},
			want:   BackupState{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError) &&
					assert.ErrorContains(t, err, "get components: collect backupable items: list component secrets")
			},
		},
		{
			name: "fail to list component PVCs",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithInterceptorFuncs(interceptor.Funcs{
					List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
						if unstructuredList, ok := list.(*unstructured.UnstructuredList); ok && unstructuredList.GetKind() == "PersistentVolumeClaimList" {
							return assert.AnError
						}
						return client.List(ctx, list, opts...)
					},
				})
			},
			backup: &backupv1.Backup{Name: testBackupName, Namespace: testNamespace},
			want:   BackupState{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError) &&
					assert.ErrorContains(t, err, "get components: collect backupable items: list component PVCs")
			},
		},
		{
			name: "fail because component name label is missing",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithInterceptorFuncs(interceptor.Funcs{
					List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
						err := client.List(ctx, list, opts...)
						if unstructuredList, ok := list.(*unstructured.UnstructuredList); ok {
							missingLabelItem := unstructured.Unstructured{}
							missingLabelItem.SetKind("PersistentVolumeClaim")
							missingLabelItem.SetLabels(map[string]string{})
							unstructuredList.Items = []unstructured.Unstructured{missingLabelItem}
						}
						return err
					},
				})
			},
			backup: &backupv1.Backup{
				Name: testBackupName, Namespace: testNamespace,
			},
			want: BackupState{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorContains(t, err, "get components: get components from items: PersistentVolumeClaim  does not have k8s.cloudogu.com/component.name label, despite being selected")
			},
		},
		{
			name: "fail because component version label is missing",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithInterceptorFuncs(interceptor.Funcs{
					List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
						err := client.List(ctx, list, opts...)
						if unstructuredList, ok := list.(*unstructured.UnstructuredList); ok {
							missingLabelItem := unstructured.Unstructured{}
							missingLabelItem.SetKind("PersistentVolumeClaim")
							missingLabelItem.SetLabels(map[string]string{"k8s.cloudogu.com/component.name": "lop-idp"})
							unstructuredList.Items = []unstructured.Unstructured{missingLabelItem}
						}
						return err
					},
				})
			},
			backup: &backupv1.Backup{
				Name: testBackupName, Namespace: testNamespace,
			},
			want: BackupState{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorContains(t, err, "get components: get components from items: PersistentVolumeClaim  does not have k8s.cloudogu.com/component.version label, despite being selected")
			},
		},
		{
			name: "fail to get global config",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						if key.Name == "global-config" {
							return assert.AnError
						}
						return client.Get(ctx, key, obj, opts...)
					},
				})
			},
			backup: &backupv1.Backup{Name: testBackupName, Namespace: testNamespace},
			want:   BackupState{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError) &&
					assert.ErrorContains(t, err, "get fqdn and domain: get global config")
			},
		},
		{
			name: "missing config.yaml key in global config",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithObjects(
					&corev1.ConfigMap{Name: "global-config", Namespace: testNamespace},
				)
			},
			backup: &backupv1.Backup{Name: testBackupName, Namespace: testNamespace},
			want:   BackupState{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorContains(t, err, "get fqdn and domain: global config does not contain config.yaml")
			},
		},
		{
			name: "missing fqdn in global config",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithObjects(
					&corev1.ConfigMap{Name: "global-config", Namespace: testNamespace, Data: map[string]string{"config.yaml": "{}"}},
				)
			},
			backup: &backupv1.Backup{Name: testBackupName, Namespace: testNamespace},
			want:   BackupState{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorContains(t, err, "get fqdn and domain: get fqdn from global config: map does not contain fqdn")
			},
		},
		{
			name: "domain is not string in global config",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithObjects(
					&corev1.ConfigMap{Name: "global-config", Namespace: testNamespace, Data: map[string]string{"config.yaml": `{fqdn: example.com, domain: {}}`}},
				)
			},
			backup: &backupv1.Backup{Name: testBackupName, Namespace: testNamespace},
			want:   BackupState{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorContains(t, err, "get fqdn and domain: get domain from global config: map value for domain is not a string")
			},
		},
		{
			name: "success with ignore v2 dogus, misc and global config",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithObjects(
					&blueprintv3.Blueprint{Name: "blueprint", Namespace: testNamespace, Spec: blueprintv3.BlueprintSpec{DisplayName: "test-blueprint"}},
					&doguv3.Dogu{Name: "cockpit", Namespace: testNamespace, Spec: doguv3.DoguSpec{Name: "cockpit", Version: "1.2.3", DoguNamespace: "official", DoguApiVersion: "v2"}},
					&doguv3.Dogu{Name: "baseline", Namespace: testNamespace, Spec: doguv3.DoguSpec{Name: "baseline", Version: "3.2.1", DoguNamespace: "official", DoguApiVersion: "v3"}},
					&corev1.ConfigMap{Name: "lop-idp-config", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "lop-idp", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "lop-idp"}},
					&corev1.Secret{Name: "lop-idp-config", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "lop-idp", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "lop-idp"}},
					&corev1.PersistentVolumeClaim{Name: "lop-idp-pvc", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "lop-idp", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "lop-idp"}},
					&corev1.ConfigMap{Name: "global-config", Namespace: testNamespace, Data: map[string]string{"config.yaml": `{"fqdn": "fqdn.example.com","domain": "domain.example.com"}`}},
				)
			},
			backup: &backupv1.Backup{
				Name: testBackupName, Namespace: testNamespace,
				Spec: backupv1.BackupSpec{
					Options: backupv1.UnitOptions{
						IgnoreV2Dogus:      true,
						IgnoreMisc:         true,
						IgnoreGlobalConfig: true,
					}},
			},
			want: BackupState{
				Name:             testBackupName,
				Coverage:         backupv1.BackupCoveragePartial,
				BlueprintId:      "test-blueprint",
				Units:            []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeDogu, Name: "baseline"}},
				InstalledDogusV2: nil,
				InstalledDogusV3: []backupv1.DoguVersion{{Name: "baseline", Version: "3.2.1"}},
				Components:       nil,
				Fqdn:             "",
				Domain:           "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "success with excluded dogus and components",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithObjects(
					&blueprintv3.Blueprint{Name: "blueprint", Namespace: testNamespace, Spec: blueprintv3.BlueprintSpec{DisplayName: "test-blueprint"}},
					&doguv3.Dogu{Name: "cockpit", Namespace: testNamespace, Spec: doguv3.DoguSpec{Name: "cockpit", Version: "1.2.3", DoguNamespace: "official", DoguApiVersion: "v2"}},
					&doguv3.Dogu{Name: "baseline", Namespace: testNamespace, Spec: doguv3.DoguSpec{Name: "baseline", Version: "3.2.1", DoguNamespace: "official", DoguApiVersion: "v3"}},
					&doguv3.Dogu{Name: "excluded", Namespace: testNamespace, Spec: doguv3.DoguSpec{Name: "excluded", Version: "3.2.1", DoguNamespace: "official", DoguApiVersion: "v3"}},
					&doguv3.Dogu{Name: "unknown-version", Namespace: testNamespace, Spec: doguv3.DoguSpec{Name: "excluded", Version: "3.2.1", DoguNamespace: "official", DoguApiVersion: "v4"}},
					&corev1.ConfigMap{Name: "lop-idp-config", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "lop-idp", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "lop-idp"}},
					&corev1.Secret{Name: "lop-idp-config", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "lop-idp", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "lop-idp"}},
					&corev1.PersistentVolumeClaim{Name: "lop-idp-pvc", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "lop-idp", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "other"}},
					&corev1.ConfigMap{Name: "excluded-config", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "excluded", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "excluded"}},
					&corev1.Secret{Name: "excluded-config", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "excluded", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "excluded"}},
					&corev1.PersistentVolumeClaim{Name: "excluded-pvc", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "excluded", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "excluded"}},
					&corev1.ConfigMap{Name: "global-config", Namespace: testNamespace, Data: map[string]string{"config.yaml": `{"fqdn": "fqdn.example.com","domain": "domain.example.com"}`}},
				)
			},
			backup: &backupv1.Backup{
				Name: testBackupName, Namespace: testNamespace,
				Spec: backupv1.BackupSpec{
					ExcludedComponents: []string{"excluded"},
					ExcludedDogus:      []string{"excluded"},
				},
			},
			want: BackupState{
				Name:        testBackupName,
				Coverage:    backupv1.BackupCoveragePartial,
				BlueprintId: "test-blueprint",
				Units: []backupv1.UnitIdentifier{
					{Type: "doguv2"},
					{Type: "dogu", Name: "baseline"},
					{Type: "global-config"},
					{Type: "misc"},
				},
				InstalledDogusV2: []backupv1.DoguVersion{{Name: "cockpit", Version: "1.2.3"}},
				InstalledDogusV3: []backupv1.DoguVersion{{Name: "baseline", Version: "3.2.1"}},
				Components:       []backupv1.ComponentStatus{{Name: "lop-idp", Version: "1.0.0"}},
				Fqdn:             "fqdn.example.com",
				Domain:           "domain.example.com",
			},
			wantErr: assert.NoError,
		},
		{
			name: "success with full coverage",
			k8sClientBuilderModifier: func(builder *fake.ClientBuilder) *fake.ClientBuilder {
				return builder.WithObjects(
					&blueprintv3.Blueprint{Name: "blueprint", Namespace: testNamespace, Spec: blueprintv3.BlueprintSpec{DisplayName: "test-blueprint"}},
					&doguv3.Dogu{Name: "cockpit", Namespace: testNamespace, Spec: doguv3.DoguSpec{Name: "cockpit", Version: "1.2.3", DoguNamespace: "official", DoguApiVersion: "v2"}},
					&doguv3.Dogu{Name: "baseline", Namespace: testNamespace, Spec: doguv3.DoguSpec{Name: "baseline", Version: "3.2.1", DoguNamespace: "official", DoguApiVersion: "v3"}},
					&doguv3.Dogu{Name: "unknown-version", Namespace: testNamespace, Spec: doguv3.DoguSpec{Name: "excluded", Version: "3.2.1", DoguNamespace: "official", DoguApiVersion: "v4"}},
					&corev1.ConfigMap{Name: "lop-idp-config", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "lop-idp", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "lop-idp"}},
					&corev1.Secret{Name: "lop-idp-config", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "lop-idp", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "lop-idp"}},
					&corev1.PersistentVolumeClaim{Name: "lop-idp-pvc", Namespace: testNamespace, Labels: map[string]string{componentv1.ComponentNameLabelKey: "lop-idp", componentv1.ComponentVersionLabelKey: "1.0.0", backupScopeLabelKey: "lop-idp"}},
					&corev1.ConfigMap{Name: "global-config", Namespace: testNamespace, Data: map[string]string{"config.yaml": `{"fqdn": "fqdn.example.com","domain": "domain.example.com"}`}},
				)
			},
			backup: &backupv1.Backup{
				Name: testBackupName, Namespace: testNamespace,
			},
			want: BackupState{
				Name:        testBackupName,
				Coverage:    backupv1.BackupCoverageFull,
				BlueprintId: "test-blueprint",
				Units: []backupv1.UnitIdentifier{
					{Type: "doguv2"},
					{Type: "dogu", Name: "baseline"},
					{Type: "global-config"},
					{Type: "misc"},
				},
				InstalledDogusV2: []backupv1.DoguVersion{{Name: "cockpit", Version: "1.2.3"}},
				InstalledDogusV3: []backupv1.DoguVersion{{Name: "baseline", Version: "3.2.1"}},
				Components:       []backupv1.ComponentStatus{{Name: "lop-idp", Version: "1.0.0"}},
				Fqdn:             "fqdn.example.com",
				Domain:           "domain.example.com",
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewCollector(tt.k8sClientBuilderModifier(fake.NewClientBuilder().WithScheme(testScheme(t))).Build())
			got, err := b.Collect(t.Context(), tt.backup)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	err := backupv1.AddToScheme(scheme)
	assert.NoError(t, err)
	err = blueprintv3.AddToScheme(scheme)
	assert.NoError(t, err)
	err = doguv3.AddToScheme(scheme)
	assert.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	assert.NoError(t, err)

	return scheme
}
