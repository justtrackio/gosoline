package cli

import "slices"

// NewInputWithRouter parses arguments using the flags of the selected command,
// including inherited group and global flags. This permits boolean switches
// before command names and preserves aliases with different kinds across commands.
func NewInputWithRouter(args []string, router *Router, globalFlags ...Flag) (*Input, error) {
	for _, scope := range inputFlagScopes(router, nil, globalFlags) {
		input, err := NewInputWithArgs(args, scope.flags...)
		if err != nil {
			continue
		}

		if scope.matches(input.Arguments) {
			return input, nil
		}
	}

	// Keep normal parsing errors and unknown-command help when no route matches.
	return NewInputWithArgs(args, globalFlags...)
}

type inputFlagScope struct {
	path      []string
	flags     []Flag
	isDefault bool
}

func (s inputFlagScope) matches(args []string) bool {
	if s.isDefault {
		return slices.Equal(args, s.path)
	}

	return len(args) >= len(s.path) && slices.Equal(args[:len(s.path)], s.path)
}

func inputFlagScopes(router *Router, path []string, inherited []Flag) []inputFlagScope {
	scopes := make([]inputFlagScope, 0)
	for _, name := range router.groupNames {
		group := router.groups[name]
		groupPath := slices.Concat(path, []string{name})
		flags := slices.Concat(inherited, group.Flags)
		scopes = append(scopes, inputFlagScopes(group.child, groupPath, flags)...)
	}

	for _, name := range router.cmdNames {
		cmd := router.cmds[name]
		scopes = append(scopes, inputFlagScope{
			path:  slices.Concat(path, []string{name}),
			flags: slices.Concat(inherited, cmd.Flags),
		})
	}

	if router.defaultCmd.Name != "" {
		scopes = append(scopes, inputFlagScope{
			path:      path,
			flags:     slices.Concat(inherited, router.defaultCmd.Flags),
			isDefault: true,
		})
	}

	return scopes
}
