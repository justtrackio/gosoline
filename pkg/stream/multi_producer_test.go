package stream_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/assert"
)

type mockConf struct {
	err        error
	expectCall bool
}

func TestMultiProducerTestCallingOfChildren(t *testing.T) {
	testErr := errors.New("testErr")

	tt := []struct {
		name      string
		mocks     []mockConf
		expectErr error
	}{
		{
			name:      "none",
			mocks:     []mockConf{},
			expectErr: nil,
		},
		{
			name: "one success",
			mocks: []mockConf{
				{nil, true},
			},
			expectErr: nil,
		},
		{
			name: "all success",
			mocks: []mockConf{
				{nil, true},
				{nil, true},
				{nil, true},
			},
			expectErr: nil,
		},
		{
			name: "last fails",
			mocks: []mockConf{
				{nil, true},
				{nil, true},
				{testErr, true},
			},
			expectErr: testErr,
		},
		{
			name: "first fails",
			mocks: []mockConf{
				{testErr, true},
				{nil, false},
				{nil, false},
			},
			expectErr: testErr,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("writeOne_%s", tc.name), func(t *testing.T) {
			// Given
			msg := newMessage()
			attributeSets := newAttributeSetsParams()
			children := setupMocks(t, tc.mocks, "WriteOne", msg, attributeSets)

			// When
			producer := stream.NewMultiProducer(children...)
			err := producer.WriteOne(t.Context(), msg, attributeSets...)

			// Then
			assert.Equal(t, tc.expectErr, err)
		})

		t.Run(fmt.Sprintf("write_%s", tc.name), func(t *testing.T) {
			// Given
			msgs := []any{newMessage(), newMessage()}
			attributeSets := newAttributeSetsParams()
			children := setupMocks(t, tc.mocks, "WriteOne", msgs, attributeSets)

			// When
			producer := stream.NewMultiProducer(children...)
			err := producer.WriteOne(t.Context(), msgs, attributeSets...)

			// Then
			assert.Equal(t, tc.expectErr, err)
		})
	}
}

func newMessage() any {
	return &stream.Message{
		Attributes: map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		},
		Body: `{"id":3,"name":"foobar"}`,
	}
}

func newAttributeSetsParams() []map[string]string {
	return []map[string]string{
		{
			"test": "foo",
		},
		{
			"test": "100",
		},
	}
}

func setupMocks(t *testing.T, confs []mockConf, method string, msgOrMsgs any, attributeSets []map[string]string) []stream.Producer {
	children := make([]stream.Producer, len(confs))
	for i, mock := range confs {
		p := new(mocks.Producer)
		if mock.expectCall {
			// hack around variadic parameter limit of mocking package
			params := newParams(t.Context(), msgOrMsgs, attributeSets)
			p.On(method, params...).Return(mock.err)
		}
		children[i] = p
	}

	return children
}

func newParams(context context.Context, model any, sets []map[string]string) []any {
	params := []any{context, model}
	for _, set := range sets {
		params = append(params, set)
	}

	return params
}
