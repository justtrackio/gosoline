package log

import (
	"os"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/pkg/errors"
)

const (
	EcsFieldsCfgKey                      = "log.ecs"
	EcsMetadataFileEnv                   = "ECS_CONTAINER_METADATA_FILE"
	EcsMetadataKeyAvailabilityZone       = "AvailabilityZone"
	EcsMetadataKeyContainerID            = "ContainerID"
	EcsMetadataKeyHostPrivateIPv4Address = "HostPrivateIPv4Address"
	EcsMetadataKeyImageName              = "ImageName"
	EcsMetadataKeyTaskDefinitionFamily   = "TaskDefinitionFamily"
	EcsMetadataKeyTaskDefinitionRevision = "TaskDefinitionRevision"
	logFieldAvailabilityZone             = "cloud.availability_zone"
	logFieldContainerID                  = "container.id"
	logFieldHostPrivateIPv4Address       = "instance.ip"
	logFieldContainerImageName           = "container.imageName"
	logFieldTaskDefinitionFamily         = "aws.task_definition.family"
	logFieldTaskDefinitionRevision       = "aws.task_definition.revision"
)

type FieldMapper struct {
	MetadataKey string `cfg:"metadata_key"`
	FieldName   string `cfg:"field_name"`
}

type EcsConfig struct {
	Fields []FieldMapper `cfg:"fields"`
}

var defaultEcsConfig = EcsConfig{
	Fields: []FieldMapper{
		{
			MetadataKey: EcsMetadataKeyAvailabilityZone,
			FieldName:   logFieldAvailabilityZone,
		},
		{
			MetadataKey: EcsMetadataKeyContainerID,
			FieldName:   logFieldContainerID,
		},
		{
			MetadataKey: EcsMetadataKeyHostPrivateIPv4Address,
			FieldName:   logFieldHostPrivateIPv4Address,
		},
		{
			MetadataKey: EcsMetadataKeyImageName,
			FieldName:   logFieldContainerImageName,
		},
		{
			MetadataKey: EcsMetadataKeyTaskDefinitionFamily,
			FieldName:   logFieldTaskDefinitionFamily,
		},
		{
			MetadataKey: EcsMetadataKeyTaskDefinitionRevision,
			FieldName:   logFieldTaskDefinitionRevision,
		},
	},
}

func DefaultEcsConfig() EcsConfig {
	return defaultEcsConfig
}

type EcsMetadata map[string]any

var (
	ecsLck      sync.Mutex
	ecsMetadata EcsMetadata
)

func ReadEcsMetadata() (EcsMetadata, error) {
	ecsLck.Lock()
	defer ecsLck.Unlock()

	if ecsMetadata != nil {
		return ecsMetadata, nil
	}

	path, ok := os.LookupEnv(EcsMetadataFileEnv)

	if path == "" || !ok {
		return nil, nil
	}

	metadata := make(EcsMetadata)

	for {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, errors.Wrap(err, "can not read ecs metadata file")
		}

		metadata = make(EcsMetadata)
		err = json.Unmarshal(data, &metadata)
		if err != nil {
			return nil, errors.Wrap(err, "can not unmarshal ecs metadata")
		}

		if status, ok := metadata["MetadataFileStatus"]; ok {
			if status.(string) == "READY" {
				break
			}
		}

		time.Sleep(1 * time.Second)
	}

	ecsMetadata = metadata

	return ecsMetadata, nil
}

func GetEcsLoggerFields(config cfg.Config) (Fields, error) {
	metadata, err := ReadEcsMetadata()
	if err != nil {
		return nil, err
	}

	if metadata == nil {
		return nil, nil
	}

	configuredEcsFields := EcsConfig{}
	config.UnmarshalKey(EcsFieldsCfgKey, &configuredEcsFields)
	m := mapx.NewMapX()

	for _, entry := range configuredEcsFields.Fields {
		m.Set(entry.FieldName, entry.MetadataKey)
	}

	fields := m.Msi()

	return fields, nil
}
