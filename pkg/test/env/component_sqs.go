package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
)

type sqsComponent struct {
	baseComponent
	binding containerBinding
}

func (c *sqsComponent) AppOptions() []application.Option {
	return []application.Option{
		application.WithConfigMap(map[string]interface{}{
			"aws_sqs_endpoint":   fmt.Sprintf("http://%s:%s", c.binding.host, c.binding.port),
			"aws_sqs_autoCreate": true,
		}),
	}
}
