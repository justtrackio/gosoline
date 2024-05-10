package appctx_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/stretchr/testify/suite"
)

func TestMetadataTestSuite(t *testing.T) {
	suite.Run(t, new(MetadataTestSuite))
}

type MetadataTestSuite struct {
	suite.Suite
}

func (s *MetadataTestSuite) TestAppend() {
	metadata := appctx.NewMetadata()

	err := metadata.Append("key", "foo")
	s.NoError(err)

	act, err := metadata.Get("key").Slice()
	s.NoError(err)
	s.Equal([]any{"foo"}, act)

	// append duplicate string
	err = metadata.Append("key", "foo")
	s.NoError(err)

	act, err = metadata.Get("key").Slice()
	s.NoError(err)
	s.Equal([]any{"foo"}, act)

	// append new string
	err = metadata.Append("key", "bar")
	s.NoError(err)

	act, err = metadata.Get("key").Slice()
	s.NoError(err)
	s.Equal([]any{"foo", "bar"}, act)
}

func (s *MetadataTestSuite) TestAppendStruct() {
	metadata := appctx.NewMetadata()

	type someStructValue struct {
		I int
		S string
	}

	err := metadata.Append("structKey", someStructValue{1, "foo"})
	s.NoError(err)

	act, err := metadata.Get("structKey").Slice()
	s.NoError(err)
	s.Equal([]any{someStructValue{1, "foo"}}, act)

	// append duplicate someStructValue
	err = metadata.Append("structKey", someStructValue{1, "foo"})
	s.NoError(err)

	act, err = metadata.Get("structKey").Slice()
	s.NoError(err)
	s.Equal([]any{someStructValue{1, "foo"}}, act)

	// append new someStructValue
	err = metadata.Append("structKey", someStructValue{1, "bar"})
	s.NoError(err)

	act, err = metadata.Get("structKey").Slice()
	s.NoError(err)
	s.Equal([]any{someStructValue{1, "foo"}, someStructValue{1, "bar"}}, act)
}

func (s *MetadataTestSuite) TestAppendStructNotComparable() {
	metadata := appctx.NewMetadata()

	type someStructValue struct {
		Sl []int
	}

	err := metadata.Append("structKey", someStructValue{[]int{1}})
	s.NoError(err)

	err = metadata.Append("structKey", someStructValue{[]int{1}})
	s.NoError(err)

	err = metadata.Append("structKey", someStructValue{[]int{2}})
	s.NoError(err)

	act, err := metadata.Get("structKey").Slice()
	s.NoError(err)
	s.Equal([]any{someStructValue{[]int{1}}, someStructValue{[]int{2}}}, act)
}
