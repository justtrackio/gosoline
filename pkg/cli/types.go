package cli

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Router struct {
	parent *Router

	groups     map[string]Group
	groupNames []string
	cmds       map[string]Cmd
	cmdNames   []string
	defaultCmd Cmd
}

func NewRouter(parent *Router) *Router {
	return &Router{
		parent: parent,
		groups: make(map[string]Group),
		cmds:   make(map[string]Cmd),
	}
}

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

func (r *Router) Cmd(cmd Cmd) *Router {
	if _, ok := r.cmds[cmd.Name]; !ok {
		r.cmdNames = append(r.cmdNames, cmd.Name)
	}

	r.cmds[cmd.Name] = cmd

	return r
}

func (r *Router) DefaultCmd(cmd Cmd) *Router {
	r.defaultCmd = cmd

	return r
}

type Group struct {
	child *Router

	Name        string
	Description string
	Flags       []Flag
	AppOptions  []application.Option
}

type Cmd struct {
	Name        string
	Description string
	Examples    []string
	Flags       []Flag
	AppOptions  []application.Option
}

type FlagKind string

const (
	FlagKindString FlagKind = "string"
	FlagKindList   FlagKind = "list"
)

type Flag struct {
	Short       string
	Long        string
	Kind        FlagKind
	CfgKey      string
	Default     string
	Description string
	AppOptions  []application.Option
}

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

type Handler = kernel.ModuleRunFunc
