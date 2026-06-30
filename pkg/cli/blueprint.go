package cli

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/application"
)

type Blueprint struct {
	Cmd        []string
	Args       []string
	Flags      []BlueprintFlag
	AppOptions []application.Option
}

type BlueprintFlag struct {
	Key        string
	CustomKey  string
	Value      any
	AppOptions []application.Option
}

func NewBlueprint(router *Router, input *Input, globalFlags ...Flag) (*Blueprint, error) {
	if err := validateRouter(router); err != nil {
		return nil, err
	}

	var selectedCmd Cmd

	cmdArgs := make([]string, 0)
	args := make([]string, 0)
	flags := append([]Flag(nil), globalFlags...)
	appOptions := make([]application.Option, 0)
	defaultCmd := router.defaultCmd

	for i, arg := range input.Arguments {
		if group, ok := router.groups[arg]; ok {
			router = group.child
			defaultCmd = router.defaultCmd
			cmdArgs = append(cmdArgs, arg)
			flags = append(flags, group.Flags...)
			appOptions = append(appOptions, group.AppOptions...)

			continue
		}

		if cmd, ok := router.cmds[arg]; ok {
			selectedCmd = cmd
			cmdArgs = append(cmdArgs, arg)
			flags = append(flags, cmd.Flags...)
			appOptions = append(appOptions, cmd.AppOptions...)

			if next := i + 1; next < len(input.Arguments) {
				args = input.Arguments[next:]
			}

			return newBlueprint(cmdArgs, args, flags, appOptions, input)
		}

		unknownCmd := append(append([]string(nil), cmdArgs...), arg)

		return nil, &cmdNotFoundError{
			cmd:      unknownCmd,
			helpPath: cmdArgs,
		}
	}

	selectedCmd = defaultCmd
	flags = append(flags, selectedCmd.Flags...)
	appOptions = append(appOptions, selectedCmd.AppOptions...)

	return newBlueprint(cmdArgs, args, flags, appOptions, input)
}

func validateRouter(router *Router) error {
	if router.defaultCmd.Name != "" && !isValidInputName(router.defaultCmd.Name) {
		return invalidCliNameError("command", router.defaultCmd.Name)
	}

	for _, group := range router.groups {
		if !isValidInputName(group.Name) {
			return invalidCliNameError("group", group.Name)
		}

		if err := validateRouter(group.child); err != nil {
			return err
		}
	}

	for _, cmd := range router.cmds {
		if !isValidInputName(cmd.Name) {
			return invalidCliNameError("command", cmd.Name)
		}
	}

	return nil
}

func invalidCliNameError(kind string, value string) error {
	return fmt.Errorf("invalid cli %s %q: must contain only alphanumeric characters, hyphens, or underscores", kind, value)
}

func newBlueprint(cmdArgs []string, args []string, flags []Flag, appOptions []application.Option, input *Input) (*Blueprint, error) {
	blueFlags := make([]BlueprintFlag, 0)
	for _, flag := range flags {
		if flag.Kind == "" {
			flag.Kind = FlagKindString
		}

		bf, err := flagParse(input.Flags, flag)
		if err != nil {
			return nil, err
		}

		if bf != nil {
			blueFlags = append(blueFlags, *bf)
		}
	}

	return &Blueprint{
		Cmd:        cmdArgs,
		Args:       args,
		Flags:      blueFlags,
		AppOptions: appOptions,
	}, nil
}

func flagParse(input []InputFlag, flag Flag) (*BlueprintFlag, error) {
	parser, ok := flagValueParsers[flag.Kind]
	if !ok {
		return nil, fmt.Errorf("unsupported cli flag kind %q for flag %q", flag.Kind, flag.Long)
	}

	val, ok := parser(input, flag)
	if !ok {
		return nil, nil
	}

	return &BlueprintFlag{
		Key:        flag.Long,
		CustomKey:  flag.CfgKey,
		Value:      val,
		AppOptions: flag.AppOptions,
	}, nil
}

func flagMatches(inputFlag InputFlag, flag Flag) bool {
	return inputFlag.Name == flag.Short || inputFlag.Name == flag.Long
}

var flagValueParsers = map[FlagKind]func(input []InputFlag, flag Flag) (any, bool){
	FlagKindList: func(input []InputFlag, flag Flag) (any, bool) {
		values := make([]string, 0)
		for _, inputFlag := range input {
			if flagMatches(inputFlag, flag) {
				values = append(values, inputFlag.Value)
			}
		}

		if len(values) > 0 {
			return values, true
		}

		if flag.Default != "" {
			return []string{flag.Default}, true
		}

		return nil, false
	},
	FlagKindString: func(input []InputFlag, flag Flag) (any, bool) {
		var val string
		var ok bool
		for _, inputFlag := range input {
			if flagMatches(inputFlag, flag) {
				val = inputFlag.Value
				ok = true
			}
		}

		if !ok {
			val = flag.Default
		}

		if val == "" {
			return nil, false
		}

		return val, true
	},
}
