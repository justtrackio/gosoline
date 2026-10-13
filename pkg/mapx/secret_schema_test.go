package mapx_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/stretchr/testify/require"
)

func TestSecretOptionPreservesMappingAndValues(t *testing.T) {
	type child struct {
		Value  string `cfg:"Value,secret,nodecode,nocast"`
		Public string `cfg:"public"`
	}
	type settings struct {
		Child *child `cfg:"child,prefix=SDK_"`
	}
	var output settings
	ms, err := mapx.NewStruct(&output, &mapx.StructSettings{FieldTag: "cfg"})
	require.NoError(t, err)
	writeSchema := ms.SecretSchema(true)
	require.True(t, writeSchema.IsSecret("SDK_Value"))
	require.False(t, writeSchema.IsSecret("SDK_public"))
	require.False(t, writeSchema.IsSecret("child.Value"))
	readSchema := ms.SecretSchema(false)
	require.True(t, readSchema.IsSecret("child.Value"))
	require.False(t, readSchema.IsSecret("SDK_Value"))
	require.NoError(t, ms.Write(mapx.NewMapX(map[string]any{"SDK_Value": "synthetic-original", "SDK_public": "public"})))
	require.Equal(t, "synthetic-original", output.Child.Value)
	values, err := ms.Read()
	require.NoError(t, err)
	require.Equal(t, "synthetic-original", values.Get("child.Value").Data())
}

func TestSecretSchemaNilPointersArraysAndRecursiveMerges(t *testing.T) {
	type row struct {
		Opaque  string `cfg:"opaque,secret"`
		Visible string `cfg:"visible"`
	}
	type recursive struct {
		Next *recursive `cfg:"next"`
		Rows [12]*row   `cfg:"rows"`
	}
	var value *recursive
	ms, err := mapx.NewStruct(value, &mapx.StructSettings{FieldTag: "cfg"})
	require.NoError(t, err)
	var schema mapx.SecretSchema
	schema.Register("root", ms.SecretSchema(true))
	schema.Register("root", ms.SecretSchema(true))
	require.True(t, schema.IsSecret("root.next.next.rows[11].opaque"))
	require.False(t, schema.IsSecret("root.next.next.rows[11].visible"))
	require.False(t, schema.IsSecret("root.next.next.rows.11.opaque"))
}

func TestSecretSchemaRegistrationsOwnIndependentBranches(t *testing.T) {
	type child struct {
		Value string `cfg:"value"`
	}
	type root struct {
		A child `cfg:"a"`
		B child `cfg:"b"`
	}
	ms, err := mapx.NewStruct(&root{}, &mapx.StructSettings{FieldTag: "cfg"})
	require.NoError(t, err)
	derived := ms.SecretSchema(true)
	var schema mapx.SecretSchema
	schema.Register("source", derived)
	schema.Register("destination", derived)
	schema.Register("source.a.value", &mapx.SecretSchema{Secret: true})
	require.True(t, schema.IsSecret("source.a.value"))
	require.False(t, schema.IsSecret("source.b.value"))
	require.False(t, schema.IsSecret("destination.a.value"))
	require.False(t, derived.IsSecret("a.value"))
}

func TestSecretSchemaRootSequencesAndBareBracketMapKeys(t *testing.T) {
	type row struct {
		Opaque string `cfg:"opaque,secret"`
		Public string `cfg:"public"`
	}
	var rows [][]row
	sequences, err := mapx.NewStruct(&rows, &mapx.StructSettings{FieldTag: "cfg"})
	require.NoError(t, err)
	require.True(t, sequences.SecretSchema(true).IsSecret("[0][1].opaque"))
	require.False(t, sequences.SecretSchema(true).IsSecret("[0][1].public"))
	var groups map[string]row
	maps, err := mapx.NewStruct(&groups, &mapx.StructSettings{FieldTag: "cfg"})
	require.NoError(t, err)
	require.True(t, maps.SecretSchema(true).IsSecret("[0].opaque"))
	require.True(t, maps.SecretSchema(true).IsSecret("[0][1].opaque"))
	require.False(t, maps.SecretSchema(true).IsSecret("[0].public"))
}

func TestConcreteSecretSchemaHandlesInterfaceCyclesAndSubtrees(t *testing.T) {
	type recursive struct {
		Opaque string `cfg:"opaque,secret"`
		Next   any    `cfg:"next"`
	}
	value := &recursive{Opaque: "synthetic-original"}
	value.Next = value
	input := map[string]any{"client": value}
	ms, err := mapx.NewStruct(&input, &mapx.StructSettings{FieldTag: "cfg"})
	require.NoError(t, err)
	schema := ms.SecretSchema(false)
	require.True(t, schema.IsSecret("client.opaque"))
	require.True(t, schema.IsSecret("client.next.opaque"))
	require.True(t, schema.At("client").IsSecret("opaque"))
	require.True(t, schema.At("client.opaque").IsSecret(""))
	require.True(t, schema.At("client.next.opaque").IsSecret(""))
	var rooted mapx.SecretSchema
	rooted.Register("rows[0]", schema)
	require.True(t, rooted.At("rows[0].client.opaque").IsSecret(""))
	require.False(t, rooted.IsSecret("rows[1].client.opaque"))
	rooted.Merge(ms.SecretSchema(false))
	require.True(t, rooted.At("rows[0].client.opaque").IsSecret(""))
	require.Same(t, value, value.Next)
	require.Equal(t, "synthetic-original", value.Opaque)
}

func TestSecretSchemaAtCombinesNestedRegistrationsWithTypeLayers(t *testing.T) {
	type envelope struct {
		Child any `cfg:"child"`
	}
	type credential struct {
		Opaque string `cfg:"opaque,secret"`
	}
	var parent envelope
	var child credential
	parentStruct, err := mapx.NewStruct(&parent, &mapx.StructSettings{FieldTag: "cfg"})
	require.NoError(t, err)
	childStruct, err := mapx.NewStruct(&child, &mapx.StructSettings{FieldTag: "cfg"})
	require.NoError(t, err)
	var schema mapx.SecretSchema
	schema.Register("root", parentStruct.SecretSchema(false))
	schema.Register("root.child", childStruct.SecretSchema(false))
	require.True(t, schema.IsSecret("root.child.opaque"))
	require.True(t, schema.At("root.child.opaque").IsSecret(""))
}
