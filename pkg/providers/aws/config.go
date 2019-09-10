package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	errorUtil "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultConfigMapName      = "cloud-resources-aws-strategies"
	DefaultConfigMapNamespace = "kube-system"

	DefaultFinalizer = "finalizers.aws.cloud-resources-operator.integreatly.org"
	DefaultRegion    = "eu-west-1"

	regionUSEast1 = "us-east-1"
	regionUSWest2 = "us-west-2"
	regionEUWest1 = "eu-west-1"

	sesSMTPEndpointUSEast1 = "email-smtp.us-east-1.amazonaws.com"
	sesSMTPEndpointUSWest2 = "email-smtp.us-west-2.amazonaws.com"
	sesSMTPEndpointEUWest1 = "email-smtp.eu-west-1.amazonaws.com"
)

type StrategyConfig struct {
	Region      string          `json:"region"`
	RawStrategy json.RawMessage `json:"strategy"`
}

//go:generate moq -out config_moq.go . ConfigManager
type ConfigManager interface {
	ReadBlobStorageStrategy(ctx context.Context, tier string) (*StrategyConfig, error)
	ReadSMTPCredentialSetStrategy(ctx context.Context, tier string) (*StrategyConfig, error)
	GetDefaultRegionSMTPServerMapping() map[string]string
}

var _ ConfigManager = (*ConfigMapConfigManager)(nil)

type ConfigMapConfigManager struct {
	configMapName      string
	configMapNamespace string
	client             client.Client
}

func NewConfigMapConfigManager(cm string, namespace string, client client.Client) *ConfigMapConfigManager {
	if cm == "" {
		cm = DefaultConfigMapName
	}
	if namespace == "" {
		namespace = DefaultConfigMapNamespace
	}
	return &ConfigMapConfigManager{
		configMapName:      cm,
		configMapNamespace: namespace,
		client:             client,
	}
}

func NewDefaultConfigMapConfigManager(client client.Client) *ConfigMapConfigManager {
	return NewConfigMapConfigManager(DefaultConfigMapName, DefaultConfigMapNamespace, client)
}

func (m *ConfigMapConfigManager) ReadBlobStorageStrategy(ctx context.Context, tier string) (*StrategyConfig, error) {
	stratCfg, err := m.getTierStrategyForProvider(ctx, string(providers.BlobStorageResourceType), tier)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get tier to strategy mapping for resource type %s", string(providers.BlobStorageResourceType))
	}
	return stratCfg, nil
}

func (m *ConfigMapConfigManager) ReadSMTPCredentialSetStrategy(ctx context.Context, tier string) (*StrategyConfig, error) {
	stratCfg, err := m.getTierStrategyForProvider(ctx, string(providers.SMTPCredentialResourceType), tier)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get tier to strategy mapping for resource type %s", string(providers.BlobStorageResourceType))
	}
	return stratCfg, nil
}

func (m *ConfigMapConfigManager) GetDefaultRegionSMTPServerMapping() map[string]string {
	return map[string]string{
		regionUSEast1: sesSMTPEndpointUSEast1,
		regionUSWest2: sesSMTPEndpointUSWest2,
		regionEUWest1: sesSMTPEndpointEUWest1,
	}
}

func (m *ConfigMapConfigManager) getTierStrategyForProvider(ctx context.Context, resourceType string, tier string) (*StrategyConfig, error) {
	cm := &v1.ConfigMap{}
	err := m.client.Get(ctx, types.NamespacedName{Name: m.configMapName, Namespace: m.configMapNamespace}, cm)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get aws strategy config map %s in namespace %s", m.configMapName, m.configMapNamespace)
	}
	rawStrategyMapping := cm.Data[string(providers.BlobStorageResourceType)]
	if rawStrategyMapping == "" {
		return nil, errorUtil.New(fmt.Sprintf("aws strategy for resource type %s is not defined", providers.BlobStorageResourceType))
	}
	var strategyMapping map[string]*StrategyConfig
	if err = json.Unmarshal([]byte(rawStrategyMapping), &strategyMapping); err != nil {
		return nil, errorUtil.Wrapf(err, "failed to unmarshal strategy mapping for resource type %s", providers.BlobStorageResourceType)
	}
	return strategyMapping[tier], nil
}