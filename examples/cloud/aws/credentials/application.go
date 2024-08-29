package credentials

import "github.com/justtrackio/gosoline/pkg/application"

func RunExampleApplication(options ...application.Option) {
	options = append([]application.Option{
		application.WithConfigFile("examples/cloud/aws/credentials/static/config.dist.yml", "yaml"),
	}, options...)

	application.New(options...).Run()
}
