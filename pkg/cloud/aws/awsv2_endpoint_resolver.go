package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
)

type endpointResolver struct {
	url string
}

func (e *endpointResolver) ResolveEndpoint(service, region string, options ...interface{}) (aws.Endpoint, error) {
	if e.url == "" {
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	}

	return aws.Endpoint{
		PartitionID:   "aws",
		URL:           e.url,
		SigningRegion: region,
	}, nil
}

func EndpointResolver(url string) aws.EndpointResolverWithOptions {
	return &endpointResolver{
		url: url,
	}
}
