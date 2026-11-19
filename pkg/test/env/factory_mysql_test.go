package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMySQLFactoryTmpfs(t *testing.T) {
	factory := &mysqlFactory{}

	t.Run("default is writable by non-root MySQL", func(t *testing.T) {
		config := factory.configureContainer(&mysqlSettings{})
		assert.Equal(t, []TmpfsSettings{{Path: "/var/lib/mysql", Mode: "1777"}}, config.Tmpfs)
	})

	t.Run("explicit settings are preserved", func(t *testing.T) {
		mounts := []TmpfsSettings{{Path: "/var/lib/mysql", Mode: "0750"}}
		settings := &mysqlSettings{ComponentContainerSettings: ComponentContainerSettings{Tmpfs: mounts}}
		config := factory.configureContainer(settings)
		assert.Equal(t, mounts, config.Tmpfs)
	})
}
