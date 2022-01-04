package cloud

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
)

/* Configuration Template for AWS Clients */
var ConfigTemplate = aws.Config{
	CredentialsChainVerboseErrors: aws.Bool(true),
	Region:                        aws.String(endpoints.EuCentral1RegionID),
	// LogLevel: aws.LogLevel(aws.LogDebug | aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors | aws.LogDebugWithHTTPBody),
	HTTPClient: &http.Client{
		Timeout: 1 * time.Minute,
	},
}
