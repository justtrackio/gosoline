package blob

import (
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
)

//go:generate mockery --name UrlBuilder
type UrlBuilder interface {
	GetAbsoluteUrl(path string) (string, error)
}

type urlBuilder struct {
	endpoint     string
	usePathStyle bool
	bucket       string
}

func NewUrlBuilder(config cfg.Config, name string) (UrlBuilder, error) {
	storeSettings := getStoreSettings(config, name)
	clientConfig := s3.GetClientConfig(config, storeSettings.ClientName)

	var err error
	var endpoint aws.Endpoint

	if endpoint, err = s3.ResolveEndpoint(config, storeSettings.ClientName); err != nil {
		return nil, fmt.Errorf("can not resolve s3 endpoint for client %s: %w", storeSettings.ClientName, err)
	}

	return &urlBuilder{
		endpoint:     endpoint.URL,
		usePathStyle: clientConfig.Settings.UsePathStyle,
		bucket:       storeSettings.Bucket,
	}, nil
}

func (b *urlBuilder) GetAbsoluteUrl(path string) (string, error) {
	var err error
	var blobUrl *url.URL

	if blobUrl, err = blobUrl.Parse(b.endpoint); err != nil {
		return "", fmt.Errorf("can not parse endpoint %s: %w", b.endpoint, err)
	}

	if b.usePathStyle {
		blobUrl = blobUrl.JoinPath(b.bucket, path)
	} else {
		blobUrl = blobUrl.JoinPath(path)
		blobUrl.Host = fmt.Sprintf("%s.%s", b.bucket, blobUrl.Host)
	}

	return blobUrl.String(), nil
}
