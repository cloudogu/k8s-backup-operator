package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	StageDevelopment        = "development"
	StageProduction         = "production"
	StageEnvVar             = "STAGE"
	namespaceEnvVar         = "NAMESPACE"
	logLevelEnvVar          = "LOG_LEVEL"
	imagePullSecretsEnvVar  = "IMAGE_PULL_SECRETS"
	backupRetryTimeLimit    = "BACKUP_RETRY_TIME_LIMIT"
	backupRequeueTimeEnvVar = "BACKUP_REQUEUE_TIME"
	backupStorageNameEnvVar = "BACKUP_STORAGE_NAME"
)

const (
	// OperatorAdditionalImagesConfigmapName contains the configmap name which consists of auxiliary yet necessary container images.
	OperatorAdditionalImagesConfigmapName = "k8s-backup-operator-additional-images"
	// OperatorImageConfigmapNameKey contains the key to retrieve this operators'
	// container image from the OperatorAdditionalImagesConfigmapName configmap.
	OperatorImageConfigmapNameKey = "operatorImage"
)

const (
	defaultBackupRequeueTimeSeconds = 5
	defaultBackupStorageName        = "default"
)

var log = ctrl.Log.WithName("config")

// OperatorConfig contains all configurable values for the dogu operator.
type OperatorConfig struct {
	// Version contains the current version of the operator
	Version *semver.Version
	// Namespace specifies the namespace that the operator is deployed to.
	Namespace string
	// ImagePullSecrets contains the secrets that are used to pull container images from external registries.
	// It is used for the creation of the backup schedule cronjob.
	ImagePullSecrets []corev1.LocalObjectReference
	// the contiues reconcilation will retry every time (default: 5s)
	RequeueTimeSeconds int
	// Name for provider backup storage name (default: 'default')
	BackupStorageName string
}

var Stage = StageProduction

func IsStageDevelopment() bool {
	return Stage == StageDevelopment
}

func GetStagePullPolicy() corev1.PullPolicy {
	pullPolicy := corev1.PullIfNotPresent
	if IsStageDevelopment() {
		pullPolicy = corev1.PullAlways
	}
	return pullPolicy
}

// NewOperatorConfig creates a new operator config by reading values from the environment variables
func NewOperatorConfig(version string) (*OperatorConfig, error) {
	configureStage()

	parsedVersion, err := semver.NewVersion(version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}
	log.Info(fmt.Sprintf("Version: [%s]", version))

	namespace, err := GetNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to read namespace: %w", err)
	}
	log.Info(fmt.Sprintf("Deploying the k8s dogu operator in namespace %s", namespace))

	imagePullSecrets, err := GetImagePullSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to read image pull secrets: %w", err)
	}
	log.Info(fmt.Sprintf("Using image pull secrets: %v", imagePullSecrets))

	backupRequeueTime, err := getBackupRequeueTimeSeconds()
	if err != nil {
		return nil, fmt.Errorf("failed to read backup requeue time: %w", err)
	}
	log.Info(fmt.Sprintf("Using backup requeue time: %v", backupRequeueTime))

	backupStorageName := getBackupStorageName()
	log.Info(fmt.Sprintf("Using backup storage name: %v", backupStorageName))

	return &OperatorConfig{
		Version:            parsedVersion,
		Namespace:          namespace,
		ImagePullSecrets:   imagePullSecrets,
		RequeueTimeSeconds: backupRequeueTime,
		BackupStorageName:  backupStorageName,
	}, nil
}

func configureStage() {
	var err error
	Stage, err = getEnvVar(StageEnvVar)
	if err != nil {
		log.Error(err, "Error reading stage environment variable. Use stage production")
	}

	if IsStageDevelopment() {
		log.Info("Starting in development mode! This is not recommended for production!")
	}
}

func GetLogLevel() (string, error) {
	logLevel, err := getEnvVar(logLevelEnvVar)
	if err != nil {
		return "", fmt.Errorf("failed to get env var [%s]: %w", logLevelEnvVar, err)
	}

	return logLevel, nil
}

func GetNamespace() (string, error) {
	namespace, err := getEnvVar(namespaceEnvVar)
	if err != nil {
		return "", fmt.Errorf("failed to get env var [%s]: %w", namespaceEnvVar, err)
	}

	return namespace, nil
}

func GetImagePullSecrets() ([]corev1.LocalObjectReference, error) {
	var secrets []corev1.LocalObjectReference
	// imagePullSecrets should be set but are not always mandatory
	envVar, found := os.LookupEnv(imagePullSecretsEnvVar)
	if !found {
		return secrets, nil
	}

	split := strings.Split(envVar, ",")
	for _, secretName := range split {
		secrets = append(secrets, corev1.LocalObjectReference{Name: secretName})
	}

	return secrets, nil
}

func getEnvVar(name string) (string, error) {
	env, found := os.LookupEnv(name)
	if !found {
		return "", fmt.Errorf("environment variable %s must be set", name)
	}
	return env, nil
}

func getBackupRequeueTimeSeconds() (int, error) {
	value, found := os.LookupEnv(backupRequeueTimeEnvVar)
	if !found || value == "" {
		return defaultBackupRequeueTimeSeconds, nil
	}

	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf(
			"parse environment variable %s: %w",
			backupRequeueTimeEnvVar,
			err,
		)
	}

	if seconds <= 0 {
		return 0, fmt.Errorf(
			"environment variable %s must be greater than zero",
			backupRequeueTimeEnvVar,
		)
	}

	return seconds, nil
}

func getBackupStorageName() string {
	value, found := os.LookupEnv(backupStorageNameEnvVar)
	if !found || value == "" {
		return defaultBackupStorageName
	}

	return value
}
