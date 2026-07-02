package cli

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

// Router stores the command and group hierarchy used to resolve CLI input.
type Router struct {
	parent *Router

	groups     map[string]Group
	groupNames []string
	cmds       map[string]Cmd
	cmdNames   []string
	defaultCmd Cmd
}

// NewRouter creates an empty router with an optional parent router for nested command groups.
func NewRouter(parent *Router) *Router {
	return &Router{
		parent: parent,
		groups: make(map[string]Group),
		cmds:   make(map[string]Cmd),
	}
}

// Group registers a command group and optional subcommands, returning the group's child router.
func (r *Router) Group(group Group, subCommands ...Cmd) *Router {
	group.child = NewRouter(r)

	for _, cmd := range subCommands {
		group.child.Cmd(cmd)
	}

	if _, ok := r.groups[group.Name]; !ok {
		r.groupNames = append(r.groupNames, group.Name)
	}

	r.groups[group.Name] = group

	return group.child
}

// Cmd registers a command on the router.
func (r *Router) Cmd(cmd Cmd) *Router {
	if _, ok := r.cmds[cmd.Name]; !ok {
		r.cmdNames = append(r.cmdNames, cmd.Name)
	}

	r.cmds[cmd.Name] = cmd

	return r
}

// DefaultCmd sets the command used when no explicit command matches the input.
func (r *Router) DefaultCmd(cmd Cmd) *Router {
	r.defaultCmd = cmd

	return r
}

// Group defines a named command namespace with shared flags and application options.
type Group struct {
	child *Router

	Name        string
	Description string
	Flags       []Flag
	AppOptions  []application.Option
}

// Cmd defines an executable command with help metadata, flags, and application options.
type Cmd struct {
	Name        string
	Description string
	Arguments   CmdArguments
	Examples    []CmdExample
	Flags       []Flag
	AppOptions  []application.Option
}

// CmdArguments identifies whether a CLI command accepts positional arguments.
type CmdArguments string

const (
	// CmdArgumentsNone documents that a command does not accept positional arguments.
	CmdArgumentsNone CmdArguments = ""
	// CmdArgumentsSingle documents that a command accepts one positional argument.
	CmdArgumentsSingle CmdArguments = "single"
	// CmdArgumentsMultiple documents that a command accepts multiple positional arguments.
	CmdArgumentsMultiple CmdArguments = "multiple"
)

// FlagKind identifies how a CLI flag value is parsed.
type FlagKind string

const (
	// FlagKindString parses a flag as a single string value where the last occurrence wins.
	FlagKindString FlagKind = "string"
	// FlagKindList parses every occurrence of a flag into a string slice.
	FlagKindList FlagKind = "list"
)

// Flag defines a supported CLI flag and how it maps into application configuration.
type Flag struct {
	Short       string
	Long        string
	Kind        FlagKind
	CfgKey      string
	Default     string
	Description string
	AppOptions  []application.Option
}

// Module builds a main application module from a typed dependency factory and command handler.
func Module[T any](fac func(ctx context.Context, config cfg.Config, logger log.Logger) (T, error), call func(m T) Handler) application.Option {
	module := func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		var err error
		var module T

		if module, err = fac(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("failed to initialize module: %w", err)
		}

		moduleFunc := call(module)

		return kernel.NewModuleFunc(moduleFunc), nil
	}

	return application.WithModuleFactory("main", module)
}

// Handler is the function executed by a CLI command module.
type Handler = kernel.ModuleRunFunc

// CmdExample describes a command example rendered in help output.
type CmdExample struct {
	Description string
	Args        string
}
