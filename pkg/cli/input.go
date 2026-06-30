package cli

import (
	"os"
	"strings"
)

// Input contains the parsed command line arguments and flags.
type Input struct {
	Arguments []string
	Flags     []InputFlag
}

type InputFlag struct {
	Name  string
	Value string
}

// NewInput parses the current process command line arguments.
func NewInput() (*Input, error) {
	return NewInputWithArgs(os.Args[1:])
}

func NewInputWithArgs(args []string) (*Input, error) {
	input := &Input{
		Arguments: make([]string, 0),
		Flags:     make([]InputFlag, 0),
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--":
			input.Arguments = append(input.Arguments, args[i+1:]...)

			return input, nil
		case !strings.HasPrefix(arg, "-"):
			if err := input.addArgument(arg); err != nil {
				return nil, err
			}
		case arg == "-":
			return nil, invalidInputError("argument", arg)
		case strings.HasPrefix(arg, "--"):
			if err := input.parseLongFlag(args, &i); err != nil {
				return nil, err
			}
		default:
			if err := input.parseShortFlags(args, &i); err != nil {
				return nil, err
			}
		}
	}

	return input, nil
}

func (i *Input) parseLongFlag(args []string, idx *int) error {
	flag := args[*idx][2:]

	if separator := strings.Index(flag, "="); separator >= 0 {
		return i.addFlag(flag[:separator], flag[separator+1:])
	}

	if next := *idx + 1; next < len(args) && isFlagValue(args[next]) {
		if err := i.addFlag(flag, args[next]); err != nil {
			return err
		}

		*idx = next

		return nil
	}

	return i.addFlag(flag, "true")
}

func (i *Input) parseShortFlags(args []string, idx *int) error {
	flag := args[*idx][1:]

	if separator := strings.Index(flag, "="); separator >= 0 {
		return i.addFlag(flag[:separator], flag[separator+1:])
	}

	isCombinedShortFlags := true
	for _, r := range flag {
		if r < 'a' || r > 'z' {
			isCombinedShortFlags = false

			break
		}
	}

	if !isCombinedShortFlags {
		return i.addFlagWithOptionalValue(args, idx, flag)
	}

	if len(flag) == 1 {
		return i.addFlagWithOptionalValue(args, idx, flag)
	}

	for _, shortFlag := range flag {
		if err := i.addFlag(string(shortFlag), "true"); err != nil {
			return err
		}
	}

	return nil
}

func (i *Input) addFlagWithOptionalValue(args []string, idx *int, flag string) error {
	if next := *idx + 1; next < len(args) && isFlagValue(args[next]) {
		if err := i.addFlag(flag, args[next]); err != nil {
			return err
		}

		*idx = next

		return nil
	}

	return i.addFlag(flag, "true")
}

func (i *Input) addArgument(arg string) error {
	i.Arguments = append(i.Arguments, arg)

	return nil
}

func (i *Input) addFlag(name string, value string) error {
	if !isValidInputName(name) {
		return invalidInputError("flag", name)
	}

	i.Flags = append(i.Flags, InputFlag{
		Name:  name,
		Value: value,
	})

	return nil
}

func isFlagValue(arg string) bool {
	return !strings.HasPrefix(arg, "-") && arg != "--"
}

func isValidInputName(name string) bool {
	if name == "" {
		return false
	}

	hasNonHyphen := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			hasNonHyphen = true
		case r >= 'A' && r <= 'Z':
			hasNonHyphen = true
		case r >= '0' && r <= '9':
			hasNonHyphen = true
		case r == '_':
			hasNonHyphen = true
		case r == '-':
		default:
			return false
		}
	}

	return hasNonHyphen
}

func invalidInputError(kind string, value string) error {
	return invalidCliNameError(kind, value)
}
