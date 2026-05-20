package cli

import (
	"github.com/justtrackio/gosoline/pkg/application"
	kernelPkg "github.com/justtrackio/gosoline/pkg/kernel"
)

type Router struct {
	parent *Router

	groups     map[string]Group
	cmds       map[string]Cmd
	defaultCmd Cmd
}

func NewRouter(parent *Router) *Router {
	return &Router{
		parent: parent,
		groups: make(map[string]Group),
		cmds:   make(map[string]Cmd),
	}
}

func (r *Router) Group(group Group) *Router {
	group.child = NewRouter(r)
	r.groups[group.Name] = group

	return group.child
}

func (r *Router) Cmd(cmd Cmd) *Router {
	r.cmds[cmd.Name] = cmd

	return r
}

func (r *Router) DefaultCmd(cmd Cmd) *Router {
	r.defaultCmd = cmd

	return r
}

type Group struct {
	child      *Router
	Name       string
	Flags      []Flag
	AppOptions []application.Option
}

type Cmd struct {
	Name          string
	Flags         []Flag
	AppOptions    []application.Option
	ModuleFactory kernelPkg.ModuleFactory
}
type Flag struct {
	Short       string
	Long        string
	CfgKey      string
	Default     string
	Description string
	AppOptions  []application.Option
}
