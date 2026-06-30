package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteHelp_CommandWithExamples(t *testing.T) {
	router := NewRouter(nil)
	router.Cmd(Cmd{
		Name:        "close",
		Description: "Close an issue.",
		Examples: []string{
			"issue close 123",
			"issue close https://gitlab.com/NAMESPACE/REPO/-/issues/123",
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

  EXAMPLES

    issue close 123
    issue close https://gitlab.com/NAMESPACE/REPO/-/issues/123

  FLAGS

    -h --help  Show help for this command.
    -R --repo  Select another repository.
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
	assert.Contains(t, buf.String(), "-i --include [--include ...]  Include a value.")
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
	assert.Contains(t, buf.String(), "-e --env")
	assert.Contains(t, buf.String(), "Select an environment. (default: dev)")
	assert.Contains(t, buf.String(), "--output")
	assert.Contains(t, buf.String(), "(default: json)")
}
