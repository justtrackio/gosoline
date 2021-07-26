package parquet

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/uuid"
	"time"
)

type S3PrefixNamingStrategy func(modelId mdl.ModelId, datetime time.Time) string
type S3KeyNamingStrategy func(modelId mdl.ModelId, datetime time.Time, prefixCallback S3PrefixNamingStrategy) string

const (
	NamingStrategyDtErrored   = "errors/yyyy/MM/dd"
	NamingStrategyDtSeparated = "yyyy/MM/dd"
)

var s3PrefixNamingStrategies = map[string]S3PrefixNamingStrategy{
	NamingStrategyDtErrored:   dtErrored,
	NamingStrategyDtSeparated: dtSeparated,
}

type ReaderSettings struct {
	ModelId        mdl.ModelId
	NamingStrategy string
	Recorder       FileRecorder
}

type S3BucketNamingStrategy func(appId cfg.AppId) string

func WithS3BucketNamingStrategy(strategy S3BucketNamingStrategy) {
	s3BucketNamingStrategy = strategy
}

var s3BucketNamingStrategy = func(appId cfg.AppId) string {
	return fmt.Sprintf("%s-%s-%s", appId.Project, appId.Environment, appId.Family)
}

func dtSeparated(modelId mdl.ModelId, datetime time.Time) string {
	return fmt.Sprintf("datalake/%s/year=%s/month=%s/day=%s", modelId.Name, datetime.Format("2006"), datetime.Format("01"), datetime.Format("02"))
}

func dtErrored(modelId mdl.ModelId, datetime time.Time) string {
	return fmt.Sprintf("datalake-errors/%s/result=format-conversion-failed/year=%s/month=%s/day=%s", modelId.Name, datetime.Format("2006"), datetime.Format("01"), datetime.Format("02"))
}

var DefaultS3KeyNamingStrategy = func(modelId mdl.ModelId, datetime time.Time, prefixCallback S3PrefixNamingStrategy) string {
	prefix := prefixCallback(modelId, datetime)
	timestamp := datetime.Format("2006-01-02-15-04-05")
	uuid := uuid.New().NewV4()

	return fmt.Sprintf("%s/%s-%s-%s-%s-%s-%s.parquet", prefix, modelId.Project, modelId.Environment, modelId.Family, modelId.Name, timestamp, uuid)
}

var s3KeyNamingStrategy = DefaultS3KeyNamingStrategy

func WithKeyNamingStrategy(strategy S3KeyNamingStrategy) {
	s3KeyNamingStrategy = strategy
}

type Partitionable interface {
	GetPartitionTimestamp() time.Time
}
