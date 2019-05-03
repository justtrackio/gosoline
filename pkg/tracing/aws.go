package tracing

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-xray-sdk-go/xray"
)

func AWS(config cfg.Config, c *client.Client) {
	enabled := config.GetBool("tracing_enabled")

	if !enabled {
		return
	}

	xray.AWS(c)
}
