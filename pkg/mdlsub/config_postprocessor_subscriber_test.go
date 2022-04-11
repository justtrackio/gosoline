package mdlsub

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func FuzzGetSubscriberConfigKey(f *testing.F) {
	testcases := []string{"Hello, world", " ", "!12345"}
	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, orig string) {
		got := GetSubscriberConfigKey(orig)

		if fmt.Sprintf("mdlsub.subscribers.%s", orig) != got {
			t.FailNow()
		}
	})
}

func FuzzGetSubscriberOutputConfigKey(f *testing.F) {
	testcases := []string{"Hello, world", " ", "!12345"}
	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, orig string) {
		got := GetSubscriberOutputConfigKey(orig)

		if fmt.Sprintf("%s.output", GetSubscriberConfigKey(orig)) != got {
			t.FailNow()
		}
	})
}

func TestGetSubscriberConfigKey(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		 {
		 	name: "first",
		 	args: "abcd",
		 	want: "mdlsub.subscribers.abcd",
		 },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSubscriberConfigKey(tt.args)

			assert.Equal(t, tt.want, got)
		})
	}
}
