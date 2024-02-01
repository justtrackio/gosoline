package httpserver

import (
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	requestSizeFields           = "goso.request.sizeFields"
	requestCompressedSizeFields = "goso.request.compressed.sizeFields"
	responseSizeFields          = "goso.response.sizeFields"
)

type sizeData struct {
	size *int
}

type encodedSizeData struct {
	sizeData
	contentEncoding string
}

func getRequestSizeFields(ginCtx *gin.Context) log.Fields {
	result := log.Fields{}

	requestSizeFields, found := ginCtx.Get(requestSizeFields)
	if found && requestSizeFields != nil {
		if data, ok := requestSizeFields.(sizeData); ok {
			result["request_bytes"] = *data.size
		}
	}

	requestCompressedSizeFields, found := ginCtx.Get(requestCompressedSizeFields)
	if found && requestCompressedSizeFields != nil {
		if data, ok := requestCompressedSizeFields.(encodedSizeData); ok {
			result["request_compression"] = data.contentEncoding
			result["request_uncompressed_bytes"] = *data.size
		}
	}

	responseSizeFields, found := ginCtx.Get(responseSizeFields)
	if found && responseSizeFields != nil {
		if data, ok := responseSizeFields.(encodedSizeData); ok {
			result["response_compression"] = data.contentEncoding
			result["uncompressed_bytes"] = *data.size
		}
	}

	return result
}
