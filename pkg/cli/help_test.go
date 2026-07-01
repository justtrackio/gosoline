package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteHelp_CommandWithExamples(t *testing.T) {
	router := NewRouter(nil)
	router.Cmd(Cmd{
		Name:        "close",
		Description: "Close an issue.",
		Examples: []CmdExample{
			{
				Description: "Close an issue by its ID:",
				Args:        "issue close 123",
			},
			{
				Description: "Close an issue by its URL:",
				Args:        "issue close https://gitlab.com/NAMESPACE/REPO/-/issues/123 \\\n    --reason completed",
			},
		},
		Flags: []Flag{
			{Short: "R", Long: "repo", Description: "Select another repository."},
		},
	})

	cli := &Cli{
		Router: router,
		name:   "glab issue",
	}

	buf := bytes.Buffer{}
	err := cli.writeHelp(&buf, "close")

	require.NoError(t, err)
	assert.Equal(t, `
  Close an issue.

  USAGE

    glab issue close [--flags]

  FLAGS

    -h, --help  Show help for this command.
    -R, --repo  Select another repository.

  EXAMPLES

    Example 1:

    Close an issue by its ID:

        issue close 123

    Example 2:

    Close an issue by its URL:

        issue close https://gitlab.com/NAMESPACE/REPO/-/issues/123 \
            --reason completed
`, buf.String())
}

func TestWriteHelp_CommandWithoutExamples(t *testing.T) {
	router := NewRouter(nil)
	router.Cmd(Cmd{
		Name:        "close",
		Description: "Close an issue.",
	})

	cli := &Cli{
		Router: router,
		name:   "glab issue",
	}

	buf := bytes.Buffer{}
	err := cli.writeHelp(&buf, "close")

	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "EXAMPLES")
}

func TestWriteHelp_DefaultLineLengthWrapsDescription(t *testing.T) {
	router := NewRouter(nil)
	router.Cmd(Cmd{
		Name:        "close",
		Description: "Close issues after validating repository permissions, recording an audit trail for later investigation, and notifying stakeholders about the resolved issue.",
	})

	cli := &Cli{
		Router: router,
		name:   "glab issue",
	}

	buf := bytes.Buffer{}
	err := cli.writeHelp(&buf, "close")

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "  Close issues after validating repository permissions, recording an audit trail for later investigation, and notifying\n")
	assert.Contains(t, buf.String(), "  stakeholders about the resolved issue.\n")
	assertHelpLinesAtMost(t, buf.String(), defaultHelpLineLength)
}

func TestWriteHelp_CustomLineLengthWrapsHelpEntries(t *testing.T) {
	router := NewRouter(nil)
	router.Cmd(Cmd{
		Name: "close",
		Flags: []Flag{
			{Short: "R", Long: "repo", Description: "Select another repository and include archived issues in the command output."},
		},
	})

	cli := &Cli{
		Router:         router,
		name:           "glab issue",
		helpLineLength: 50,
	}

	buf := bytes.Buffer{}
	err := cli.writeHelp(&buf, "close")

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "    -R, --repo  Select another repository and\n")
	assert.Contains(t, buf.String(), "                include archived issues in the\n")
	assert.Contains(t, buf.String(), "                command output.\n")
	assertHelpLinesAtMost(t, buf.String(), 50)
}

func TestWriteHelp_ExampleDescriptionWrapsButArgsDoNot(t *testing.T) {
	longArgs := "issue close https://gitlab.com/NAMESPACE/REPO/-/issues/123 --reason completed --notify-watchers"
	router := NewRouter(nil)
	router.Cmd(Cmd{
		Name: "close",
		Examples: []CmdExample{
			{
				Description: "Close a single issue by its URL and assign a detailed resolution reason.",
				Args:        longArgs,
			},
		},
	})

	cli := &Cli{
		Router:         router,
		name:           "glab issue",
		helpLineLength: 50,
	}

	buf := bytes.Buffer{}
	err := cli.writeHelp(&buf, "close")

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "    Close a single issue by its URL and assign a\n")
	assert.Contains(t, buf.String(), "    detailed resolution reason.\n")
	assert.Contains(t, buf.String(), "        "+longArgs+"\n")
}

func TestWriteHelp_ListFlag(t *testing.T) {
	router := NewRouter(nil)
	router.Cmd(Cmd{
		Name: "run",
		Flags: []Flag{
			{Short: "i", Long: "include", Kind: FlagKindList, Description: "Include a value."},
		},
	})

	cli := &Cli{
		Router: router,
		name:   "app",
	}

	buf := bytes.Buffer{}
	err := cli.writeHelp(&buf, "run")

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "-i, --include [--include ...]  Include a value.")
}

func TestWriteHelp_LongOnlyListFlag(t *testing.T) {
	router := NewRouter(nil)
	router.Cmd(Cmd{
		Name: "run",
		Flags: []Flag{
			{Long: "include", Kind: FlagKindList, Description: "Include a value."},
		},
	})

	cli := &Cli{
		Router: router,
		name:   "app",
	}

	buf := bytes.Buffer{}
	err := cli.writeHelp(&buf, "run")

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "--include [--include ...]  Include a value.")
}

func TestWriteHelp_FlagDefault(t *testing.T) {
	router := NewRouter(nil)
	router.Cmd(Cmd{
		Name: "run",
		Flags: []Flag{
			{Short: "e", Long: "env", Default: "dev", Description: "Select an environment."},
			{Long: "output", Default: "json"},
		},
	})

	cli := &Cli{
		Router: router,
		name:   "app",
	}

	buf := bytes.Buffer{}
	err := cli.writeHelp(&buf, "run")

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "-e, --env")
	assert.Contains(t, buf.String(), "Select an environment. (default: dev)")
	assert.Contains(t, buf.String(), "--output")
	assert.Contains(t, buf.String(), "(default: json)")
}

func assertHelpLinesAtMost(t *testing.T, output string, maxLength int) {
	t.Helper()

	for _, line := range strings.Split(output, "\n") {
		assert.LessOrEqual(t, len(line), maxLength, "line %q exceeds configured help line length", line)
	}
}
