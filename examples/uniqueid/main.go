package main

import (
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/uniqueid"
)

func main() {
	app := application.Default()
	app.Add("api", apiserver.New(uniqueid.DefineApi))
	app.Run()
}
