package env

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMySQLFactoryTmpfs(t *testing.T) {
	factory := &mysqlFactory{}
	runner := &containerRunnerLocal{}

	t.Run("default data directory is private", func(t *testing.T) {
		config := factory.configureContainer(&mysqlSettings{})
		assert.Equal(t, map[string]string{"/var/lib/mysql": "mode=0700"}, runner.getTmpfsConfig(config.Tmpfs))
	})

	t.Run("data directory owner from environment", func(t *testing.T) {
		t.Setenv("TEST_COMPONENTS_MYSQL_DEFAULT_TMPFS_0_UID", "1234")
		t.Setenv("TEST_COMPONENTS_MYSQL_DEFAULT_TMPFS_0_GID", "0")

		settings := &mysqlSettings{}
		conf := cfg.New()
		require.NoError(t, conf.Option(
			cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
			cfg.WithConfigSetting("test.components.mysql.default.tmpfs[0].uid", 4321),
		))
		require.NoError(t, UnmarshalSettings(conf, settings, componentMySql, "default"))
		config := factory.configureContainer(settings)
		assert.Equal(t, map[string]string{"/var/lib/mysql": "mode=0700,uid=1234,gid=0"}, runner.getTmpfsConfig(config.Tmpfs))
	})

	t.Run("explicit settings are preserved", func(t *testing.T) {
		mounts := []TmpfsSettings{{Path: "/custom/mysql", Size: "64m", Mode: "0750"}}
		settings := &mysqlSettings{ComponentContainerSettings: ComponentContainerSettings{Tmpfs: mounts}}
		config := factory.configureContainer(settings)
		assert.Equal(t, map[string]string{"/custom/mysql": "size=64m,mode=0750"}, runner.getTmpfsConfig(config.Tmpfs))
	})
}
