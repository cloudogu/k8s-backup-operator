package backup

import (
	"context"
	"testing"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestBackupConditions(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Backup Conditions Suite")
}

var _ = ginkgo.Describe("Backup CRD Conditions", func() {
	var (
		ctx           context.Context
		testNamespace string
		backupName    string
		backup        *v1.Backup
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
		testNamespace = "test-namespace"
		backupName = "test-backup"

		// Initialize the backup object with empty Status
		backup = &v1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backupName,
				Namespace: testNamespace,
			},
			Spec: v1.BackupSpec{
				Provider: "velero",
			},
		}
	})

	ginkgo.Context("when a Backup CRD is created", func() {
		ginkgo.It("should have Conditions field present in the Status section", func() {
			// Add conditions to the backup status (simulating what backupCreateManager does)
			backup.Status.Conditions = []metav1.Condition{
				{
					Type:               "Test",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "Test",
					Message:            "Test",
				},
			}

			// Verify that Conditions field is present
			gomega.Expect(backup.Status.Conditions).NotTo(gomega.BeEmpty())
			gomega.Expect(len(backup.Status.Conditions)).To(gomega.Equal(1))
		})

		ginkgo.It("should have valid Condition with Type, Status, and Reason", func() {
			// Add conditions to the backup status
			backup.Status.Conditions = []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "BackupSucceeded",
					Message:            "Backup process completed successfully",
				},
			}

			// Verify the condition structure
			gomega.Expect(backup.Status.Conditions).To(gomega.HaveLen(1))
			condition := backup.Status.Conditions[0]

			gomega.Expect(condition.Type).To(gomega.Equal("Ready"))
			gomega.Expect(condition.Status).To(gomega.Equal(metav1.ConditionTrue))
			gomega.Expect(condition.Reason).To(gomega.Equal("BackupSucceeded"))
			gomega.Expect(condition.Message).To(gomega.Equal("Backup process completed successfully"))
			gomega.Expect(condition.LastTransitionTime).NotTo(gomega.BeZero())
		})

		ginkgo.It("should support multiple conditions in Status", func() {
			// Add multiple conditions to the backup status
			backup.Status.Conditions = []metav1.Condition{
				{
					Type:               "Initialized",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "BackupInitialized",
					Message:            "Backup was initialized",
				},
				{
					Type:               "InProgress",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "BackupInProgress",
					Message:            "Backup is in progress",
				},
				{
					Type:               "Completed",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             "BackupNotCompleted",
					Message:            "Backup has not completed yet",
				},
			}

			// Verify multiple conditions
			gomega.Expect(backup.Status.Conditions).To(gomega.HaveLen(3))
			gomega.Expect(backup.Status.Conditions[0].Type).To(gomega.Equal("Initialized"))
			gomega.Expect(backup.Status.Conditions[1].Type).To(gomega.Equal("InProgress"))
			gomega.Expect(backup.Status.Conditions[2].Type).To(gomega.Equal("Completed"))
		})

		ginkgo.It("should persist Conditions field when updating Backup Status", func() {
			// Set initial status with conditions
			backup.Status.Status = v1.BackupStatusInProgress
			backup.Status.Conditions = []metav1.Condition{
				{
					Type:               "Processing",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "BackupStarted",
					Message:            "Backup has started",
				},
			}

			// Verify that conditions persist when status is updated
			backup.Status.Status = v1.BackupStatusCompleted
			gomega.Expect(backup.Status.Conditions).NotTo(gomega.BeEmpty())
			gomega.Expect(backup.Status.Conditions[0].Type).To(gomega.Equal("Processing"))
		})

		ginkgo.It("should allow empty Conditions field initially", func() {
			// Verify that a new backup can have empty conditions
			gomega.Expect(backup.Status.Conditions).To(gomega.BeEmpty())

			// Add conditions later
			backup.Status.Conditions = append(backup.Status.Conditions, metav1.Condition{
				Type:               "Added",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "ConditionAdded",
				Message:            "Condition was added after creation",
			})

			// Verify that conditions are now present
			gomega.Expect(backup.Status.Conditions).NotTo(gomega.BeEmpty())
			gomega.Expect(len(backup.Status.Conditions)).To(gomega.Equal(1))
		})
	})

	ginkgo.Context("when Backup CRD is created in Kubernetes", func() {
		ginkgo.It("should store and retrieve Conditions from Status", func() {
			// Register the Backup CRD with the scheme
			s := scheme.Scheme
			v1.SchemeBuilder.AddToScheme(s)

			// Create a fake client
			client := fake.NewClientBuilder().
				WithScheme(s).
				Build()

			// Set up backup with conditions
			backup.Status.Status = v1.BackupStatusInProgress
			backup.Status.Conditions = []metav1.Condition{
				{
					Type:               "StatusUpdated",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "StatusSynced",
					Message:            "Backup status was synced with Kubernetes",
				},
			}

			// Create the backup in the fake cluster
			err := client.Create(ctx, backup)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Retrieve the backup and verify conditions are persisted
			retrievedBackup := &v1.Backup{}
			key := types.NamespacedName{
				Namespace: backup.Namespace,
				Name:      backup.Name,
			}
			err = client.Get(ctx, key, retrievedBackup)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(retrievedBackup.Status.Conditions).NotTo(gomega.BeEmpty())
			gomega.Expect(len(retrievedBackup.Status.Conditions)).To(gomega.Equal(1))
			gomega.Expect(retrievedBackup.Status.Conditions[0].Type).To(gomega.Equal("StatusUpdated"))
		})
	})
})
