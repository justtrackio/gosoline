package cfg

import (
	"reflect"

	"github.com/justtrackio/gosoline/pkg/mapx"
)

func (c *config) IsSecret(selector string) bool {
	c.secretMu.RLock()
	defer c.secretMu.RUnlock()

	return c.secretSchema.IsSecret(selector)
}

func (c *config) registerSecrets(key string, value any, forWrite bool) {
	t := reflect.TypeOf(value)
	if t == nil {
		return
	}
	if t.Kind() != reflect.Pointer {
		value = reflect.New(t).Interface()
	}
	ms, err := mapx.NewStruct(value, &mapx.StructSettings{FieldTag: "cfg"})
	if err != nil {
		return
	}
	schema := ms.SecretSchema(forWrite)
	if forWrite {
		schema.Merge(ms.SecretSchema(false))
	}
	c.registerSecretSchema(key, schema)
}

func (c *config) registerSecretSchema(key string, schema *mapx.SecretSchema) {
	c.secretMu.Lock()
	defer c.secretMu.Unlock()
	c.secretSchema.Register(key, schema)
}

type defaultsConfig struct {
	GosoConf
	owner  *config
	schema *mapx.SecretSchema
}

func (c *config) defaultsContext(ms *mapx.Struct) Config {
	schema := ms.SecretSchema(true)
	schema.Merge(ms.SecretSchema(false))

	return &defaultsConfig{GosoConf: c, owner: c, schema: schema}
}
