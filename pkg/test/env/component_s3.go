package env

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
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

func (c *S3Component) Client() *s3.S3 {
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:         aws.String(c.s3Address),
		MaxRetries:       aws.Int(0),
		Region:           aws.String(endpoints.EuCentral1RegionID),
		Credentials:      gosoAws.GetDefaultCredentials(),
		S3ForcePathStyle: aws.Bool(true),
	}))

	return s3.New(sess)
}
