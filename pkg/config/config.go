package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	StageDevelopment       = "development"
	StageProduction        = "production"
	StageEnvVar            = "STAGE"
	namespaceEnvVar        = "NAMESPACE"
	logLevelEnvVar         = "LOG_LEVEL"
	imagePullSecretsEnvVar = "IMAGE_PULL_SECRETS"
)

const (
	// OperatorAdditionalImagesConfigmapName contains the configmap name which consists of auxiliary yet necessary container images.
	OperatorAdditionalImagesConfigmapName = "k8s-backup-operator-additional-images"
	// OperatorImageConfigmapNameKey contains the key to retrieve this operators'
	// container image from the OperatorAdditionalImagesConfigmapName configmap.
	OperatorImageConfigmapNameKey = "operatorImage"
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

	return &OperatorConfig{
		Version:          parsedVersion,
		Namespace:        namespace,
		ImagePullSecrets: imagePullSecrets,
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
