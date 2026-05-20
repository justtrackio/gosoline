package cli

import (
	"os"
	"strings"
)

// Input contains the parsed command line arguments and flags.
type Input struct {
	Arguments []string
	Flags     map[string]string
}

// NewInput parses the current process command line arguments.
func NewInput() *Input {
	return NewInputWithArgs(os.Args[1:])
}

func NewInputWithArgs(args []string) *Input {
	input := &Input{
		Arguments: make([]string, 0),
		Flags:     make(map[string]string),
	}

	parseFlags := true

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if !parseFlags {
			input.Arguments = append(input.Arguments, arg)

			continue
		}

		switch {
		case arg == "--":
			parseFlags = false
		case !strings.HasPrefix(arg, "-") || arg == "-":
			input.Arguments = append(input.Arguments, arg)
		case strings.HasPrefix(arg, "--"):
			if !input.parseLongFlag(args, &i) {
				input.Arguments = append(input.Arguments, arg)
			}
		default:
			input.parseShortFlags(args, &i)
		}
	}

	return input
}

func (i *Input) parseLongFlag(args []string, idx *int) bool {
	flag := args[*idx][2:]
	if flag == "" {
		return false
	}

	if separator := strings.Index(flag, "="); separator >= 0 {
		i.Flags[flag[:separator]] = flag[separator+1:]

		return true
	}

	if next := *idx + 1; next < len(args) && isFlagValue(args[next]) {
		i.Flags[flag] = args[next]
		*idx = next

		return true
	}

	i.Flags[flag] = "true"

	return true
}

func (i *Input) parseShortFlags(args []string, idx *int) {
	flag := args[*idx][1:]

	if len(flag) == 1 {
		if next := *idx + 1; next < len(args) && isFlagValue(args[next]) {
			i.Flags[flag] = args[next]
			*idx = next

			return
		}

		i.Flags[flag] = "true"

		return
	}

	for _, shortFlag := range flag {
		i.Flags[string(shortFlag)] = "true"
	}
}

func isFlagValue(arg string) bool {
	return arg == "-" || (!strings.HasPrefix(arg, "-") && arg != "--")
}
