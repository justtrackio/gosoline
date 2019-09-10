package parquet

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	uuid2 "github.com/twinj/uuid"
	"time"
)

type Settings struct {
	ModelId mdl.ModelId
}

type S3BucketNamingStrategy func(appId cfg.AppId) string

func WithS3BucketNamingStrategy(strategy S3BucketNamingStrategy) {
	s3BucketNamingStrategy = strategy
}

var s3BucketNamingStrategy = func(appId cfg.AppId) string {
	return fmt.Sprintf("%s-%s-%s", appId.Project, appId.Environment, appId.Family)
}

var s3PrefixNamingStrategy = func(modelId mdl.ModelId, datetime time.Time) string {
	return fmt.Sprintf("datalake/%s/dt=%s", modelId.Name, datetime.Format("20060102"))
}

var s3KeyNamingStrategy = func(modelId mdl.ModelId, datetime time.Time) string {
	prefix := s3PrefixNamingStrategy(modelId, datetime)
	timestamp := datetime.Format("2006-01-02-15-04-05")
	uuid := uuid2.NewV4().String()

	return fmt.Sprintf("%s/%s-%s-%s-%s-%s-%s.parquet", prefix, modelId.Project, modelId.Environment, modelId.Family, modelId.Name, timestamp, uuid)
}

type TimeStampable interface {
	GetCreatedAt() time.Time
}
