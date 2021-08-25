package env

import (
	"github.com/applike/gosoline/pkg/cfg"
)

type S3Component struct {
	baseComponent
	s3Address string
}

func (c *S3Component) CfgOptions() []cfg.Option {
	return []cfg.Option{
		cfg.WithConfigMap(map[string]interface{}{
			"aws_s3_endpoint":       c.s3Address,
			"aws_s3_autoCreate":     true,
			"aws_s3_forcePathStyle": true,
		}),
	}
}
