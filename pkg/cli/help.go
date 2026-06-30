package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

func (c *Cli) writeCmdNotFound(errW io.Writer, helpW io.Writer, err *cmdNotFoundError) error {
	if err := fprintf(errW, "%s\n", err); err != nil {
		return err
	}

	return c.writeHelp(helpW, err.helpPath...)
}

type helpEntry struct {
	usage       string
	description string
}

type helpContext struct {
	router      *Router
	path        []string
	cmd         *Cmd
	description string
	flags       []Flag
}

func (c *Cli) writeHelp(w io.Writer, args ...string) error {
	ctx, found := c.findHelpContext(args)
	if !found {
		return fmt.Errorf("unknown command %q", strings.Join(args, " "))
	}

	if err := writeSection(w, ctx.description); err != nil {
		return err
	}

	if err := writeUsage(w, c.name, ctx.path, ctx.cmd); err != nil {
		return err
	}

	if err := writeExamples(w, ctx.cmd); err != nil {
		return err
	}

	if ctx.cmd == nil {
		if err := writeCommands(w, ctx.router); err != nil {
			return err
		}
	}

	if err := writeFlags(w, append(c.flags, ctx.flags...)); err != nil {
		return err
	}

	return nil
}

func (c *Cli) findHelpContext(args []string) (helpContext, bool) {
	router := c.Router
	path := make([]string, 0, len(args))
	flags := make([]Flag, 0)
	description := c.description

	for _, arg := range args {
		if group, ok := router.groups[arg]; ok {
			router = group.child
			path = append(path, arg)
			flags = append(flags, group.Flags...)
			description = group.Description

			continue
		}

		if cmd, ok := router.cmds[arg]; ok {
			path = append(path, arg)

			return helpContext{
				router:      router,
				path:        path,
				cmd:         &cmd,
				description: cmd.Description,
				flags:       append(flags, cmd.Flags...),
			}, true
		}

		return helpContext{}, false
	}

	return helpContext{
		router:      router,
		path:        path,
		description: description,
		flags:       flags,
	}, true
}

func writeSection(w io.Writer, text string) error {
	if text == "" {
		return nil
	}

	return fprintf(w, "\n  %s\n", text)
}

func writeUsage(w io.Writer, name string, path []string, cmd *Cmd) error {
	usageParts := []string{name}
	usageParts = append(usageParts, path...)

	if cmd == nil {
		usageParts = append(usageParts, "<command>", "[--flags]")
	} else if len(cmd.Flags) > 0 {
		usageParts = append(usageParts, "[--flags]")
	}

	return fprintf(w, "\n  USAGE\n\n    %s\n", strings.Join(usageParts, " "))
}

func writeExamples(w io.Writer, cmd *Cmd) error {
	if cmd == nil || len(cmd.Examples) == 0 {
		return nil
	}

	if err := fprintf(w, "\n  EXAMPLES\n\n"); err != nil {
		return err
	}

	for _, example := range cmd.Examples {
		if err := fprintf(w, "    %s\n", example); err != nil {
			return err
		}
	}

	return nil
}

func writeCommands(w io.Writer, router *Router) error {
	entries := helpEntries(router)
	if len(entries) == 0 {
		return nil
	}

	width := maxHelpEntryWidth(entries)

	if err := fprintf(w, "\n  COMMANDS\n\n"); err != nil {
		return err
	}

	for _, entry := range entries {
		if err := fprintf(w, "    %-*s  %s\n", width, entry.usage, entry.description); err != nil {
			return err
		}
	}

	return nil
}

func helpEntries(router *Router) []helpEntry {
	entries := make([]helpEntry, 0, len(router.groupNames)+len(router.cmdNames))

	groupNames := append([]string(nil), router.groupNames...)
	sort.Strings(groupNames)
	for _, name := range groupNames {
		group := router.groups[name]
		entries = append(entries, helpEntry{
			usage:       fmt.Sprintf("%s <command> [--flags]", group.Name),
			description: group.Description,
		})
	}

	cmdNames := append([]string(nil), router.cmdNames...)
	sort.Strings(cmdNames)
	for _, name := range cmdNames {
		cmd := router.cmds[name]
		entries = append(entries, helpEntry{
			usage:       commandUsage(cmd),
			description: cmd.Description,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].usage < entries[j].usage
	})

	return entries
}

func commandUsage(cmd Cmd) string {
	if len(cmd.Flags) == 0 {
		return cmd.Name
	}

	return fmt.Sprintf("%s [--flags]", cmd.Name)
}

func writeFlags(w io.Writer, flags []Flag) error {
	entries := flagEntries(flags)
	if len(entries) == 0 {
		return nil
	}

	width := maxHelpEntryWidth(entries)

	if err := fprintf(w, "\n  FLAGS\n\n"); err != nil {
		return err
	}

	for _, entry := range entries {
		if err := fprintf(w, "    %-*s  %s\n", width, entry.usage, entry.description); err != nil {
			return err
		}
	}

	return nil
}

func flagEntries(flags []Flag) []helpEntry {
	entries := make([]helpEntry, 0, len(flags)+1)
	entries = append(entries, helpEntry{
		usage:       "-h --help",
		description: "Show help for this command.",
	})

	for _, flag := range flags {
		entries = append(entries, helpEntry{
			usage:       flagUsage(flag),
			description: flagDescription(flag),
		})
	}

	return entries
}

func flagDescription(flag Flag) string {
	if flag.Default == "" {
		return flag.Description
	}

	if flag.Description == "" {
		return fmt.Sprintf("(default: %s)", flag.Default)
	}

	return fmt.Sprintf("%s (default: %s)", flag.Description, flag.Default)
}

func flagUsage(flag Flag) string {
	parts := make([]string, 0, 2)

	if flag.Short != "" {
		parts = append(parts, "-"+flag.Short)
	}

	if flag.Long != "" {
		parts = append(parts, "--"+flag.Long)
	}

	if flag.Kind == FlagKindList && flag.Long != "" {
		parts = append(parts, fmt.Sprintf("[--%s ...]", flag.Long))
	}

	return strings.Join(parts, " ")
}

func maxHelpEntryWidth(entries []helpEntry) int {
	width := 0
	for _, entry := range entries {
		if len(entry.usage) > width {
			width = len(entry.usage)
		}
	}

	return width
}

func hasHelpFlag(input *Input) bool {
	for _, flag := range input.Flags {
		if flag.Name == "help" || flag.Name == "h" {
			return true
		}
	}

	return false
}

func trimHelpCommand(args []string) []string {
	if len(args) == 0 || args[0] != "help" {
		return args
	}

	return args[1:]
}

func fprintf(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)

	return err
}
