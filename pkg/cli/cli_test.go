package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlueprint_RootUnknownCommandError(t *testing.T) {
	router := NewRouter(nil)
	router.Cmd(Cmd{Name: "serve"})

	bp, err := NewBlueprint(router, &Input{Arguments: []string{"unknown"}})

	require.Error(t, err)
	assert.Nil(t, bp)
	cmdErr := &cmdNotFoundError{}
	require.ErrorAs(t, err, &cmdErr)
	assert.Equal(t, []string{"unknown"}, cmdErr.cmd)
	assert.Empty(t, cmdErr.helpPath)
	assert.EqualError(t, err, `unknown command "unknown"`)
}

func TestNewBlueprint_GroupUnknownCommandError(t *testing.T) {
	router := NewRouter(nil)
	apiGroup := router.Group(Group{Name: "api"})
	apiGroup.Cmd(Cmd{Name: "serve"})

	bp, err := NewBlueprint(router, &Input{Arguments: []string{"api", "unknown"}})

	require.Error(t, err)
	assert.Nil(t, bp)
	cmdErr := &cmdNotFoundError{}
	require.ErrorAs(t, err, &cmdErr)
	assert.Equal(t, []string{"api", "unknown"}, cmdErr.cmd)
	assert.Equal(t, []string{"api"}, cmdErr.helpPath)
	assert.EqualError(t, err, `unknown command "api unknown"`)
}

func TestWriteCmdNotFound_RootHelp(t *testing.T) {
	router := NewRouter(nil)
	router.Cmd(Cmd{Name: "serve", Description: "Serve requests."})

	cli := &Cli{
		Router:      router,
		name:        "app",
		description: "Run the app.",
	}
	errBuf := bytes.Buffer{}
	helpBuf := bytes.Buffer{}

	err := cli.writeCmdNotFound(&errBuf, &helpBuf, &cmdNotFoundError{cmd: []string{"unknown"}})

	require.NoError(t, err)
	assert.Equal(t, "unknown command \"unknown\"\n", errBuf.String())
	assert.Contains(t, helpBuf.String(), "Run the app.")
	assert.Contains(t, helpBuf.String(), "app <command> [--flags]")
	assert.Contains(t, helpBuf.String(), "serve  Serve requests.")
}

func TestWriteCmdNotFound_GroupHelp(t *testing.T) {
	router := NewRouter(nil)
	apiGroup := router.Group(Group{Name: "api", Description: "Manage the API."})
	apiGroup.Cmd(Cmd{Name: "serve", Description: "Serve requests."})

	cli := &Cli{
		Router: router,
		name:   "app",
	}
	errBuf := bytes.Buffer{}
	helpBuf := bytes.Buffer{}

	err := cli.writeCmdNotFound(&errBuf, &helpBuf, &cmdNotFoundError{
		cmd:      []string{"api", "unknown"},
		helpPath: []string{"api"},
	})

	require.NoError(t, err)
	assert.Equal(t, "unknown command \"api unknown\"\n", errBuf.String())
	assert.Contains(t, helpBuf.String(), "Manage the API.")
	assert.Contains(t, helpBuf.String(), "app api <command> [--flags]")
	assert.Contains(t, helpBuf.String(), "serve  Serve requests.")
}
