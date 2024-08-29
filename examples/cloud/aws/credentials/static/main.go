package main

import (
	"github.com/justtrackio/gosoline/examples/cloud/aws/credentials"
	"github.com/justtrackio/gosoline/pkg/application"
)

func main() {
	credentials.RunExampleApplication(
		application.WithModuleFactory("dumper", credentials.NewDebugModule),
	)
}
