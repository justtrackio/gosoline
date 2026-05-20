package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cli"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	root := cli.NewCli()

	root.Cmd(cli.Cmd{
		Name:          "test",
		ModuleFactory: cliDebug,
	})

	root.Cmd(cli.Cmd{
		Name: "flagged",
		Flags: []cli.Flag{
			{Short: "s", Long: "short", CfgKey: "", Default: "Hubert", Description: ""},
		},
		ModuleFactory: cliDebug,
	})

	root.Run()
}

var cliDebug = cli.WithRunFunc(func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.ModuleRunFunc, error) {
	return func(ctx context.Context) error {
		var err error
		var flags map[string]any

		if flags, err = config.GetStringMap("cli"); err != nil {
			return err
		}

		fmt.Println(flags)

		return nil
	}, nil
})
