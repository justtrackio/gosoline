package cli

import "github.com/justtrackio/gosoline/pkg/application"

type Blueprint struct {
	Cmd   []string
	Args  []string
	Flags []BlueprintFlag
}

type BlueprintFlag struct {
	Key        string
	CustomKey  string
	Value      string
	AppOptions []application.Option
}

func NewBlueprint(router *Router, input *Input) *Blueprint {
	var i int
	var arg string
	var cmdArgs []string
	var flags []Flag
	var selected bool

	defaultCmd := router.defaultCmd

	for i, arg = range input.Arguments {
		if group, ok := router.groups[arg]; ok {
			router = group.child
			defaultCmd = router.defaultCmd

			flags = append(flags, group.Flags...)
			cmdArgs = append(cmdArgs, arg)

			continue
		}

		if cmd, ok := router.cmds[arg]; ok {
			selected = true
			flags = append(flags, cmd.Flags...)
			cmdArgs = append(cmdArgs, arg)

			break
		}
	}

	if !selected {
		flags = append(flags, defaultCmd.Flags...)
	}

	i++
	args := make([]string, 0)
	if i < len(input.Arguments) {
		args = input.Arguments[i:]
	}

	blueFlags := make([]BlueprintFlag, 0)
	for _, flag := range flags {
		if bf := parseFlag(input.Flags, flag); bf != nil {
			blueFlags = append(blueFlags, *bf)
		}
	}

	return &Blueprint{
		Cmd:   cmdArgs,
		Args:  args,
		Flags: blueFlags,
	}
}

func parseFlag(input map[string]string, flag Flag) *BlueprintFlag {
	var ok bool
	var val string

	val, ok = input[flag.Short]

	if !ok {
		val, ok = input[flag.Long]
	}

	if !ok {
		val = flag.Default
	}

	if val == "" {
		return nil
	}

	return &BlueprintFlag{
		Key:        flag.Long,
		CustomKey:  flag.CfgKey,
		Value:      val,
		AppOptions: flag.AppOptions,
	}
}
