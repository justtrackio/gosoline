package assert

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func ElasticDocumentExists(t *testing.T, index string, id string) {
	path := fmt.Sprintf("http://localhost:9222/%s/_doc/%s", index, id)
	resp, err := http.Get(path)

	if err != nil {
		assert.Fail(t, "could not get elasticsearch document: %s", err.Error())
		return
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode, "elasticsearch should response with http ok")
}
