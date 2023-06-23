package apiserver

import (
	"compress/gzip"
	"fmt"
	"strconv"

	ginGzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// CompressionSettings control gzip support for requests and responses. By default, compressed requests are accepted and compressed responses are returned (if accepted by the client).
type CompressionSettings struct {
	Level         string `cfg:"level" default:"default" validate:"oneof=none default best fast 0 1 2 3 4 5 6 7 8 9"`
	Decompression bool   `cfg:"decompression" default:"true"`
	// Exclude files by path, extension, or regular expression from being considered for compression. Useful if you are serving a format unknown to Gosoline.
	Exclude CompressionExcludeSettings `cfg:"exclude"`
}

// CompressionExcludeSettings allow enabling of gzip support.
type CompressionExcludeSettings struct {
	Extension []string `cfg:"extension"`
	Path      []string `cfg:"path"`
	PathRegex []string `cfg:"path_regex"`
}

func configureCompression(settings CompressionSettings) ([]gin.HandlerFunc, error) {
	middlewares := make([]gin.HandlerFunc, 0)

	// we always record the request size
	middlewares = append(middlewares, recordRequestSize)

	level, err := parseLevel(settings.Level)
	if err != nil {
		return nil, err
	}

	if level == gzip.NoCompression && !settings.Decompression {
		// there is no use in adding a handler if we should neither compress nor decompress
		return middlewares, nil
	}

	opts := make([]ginGzip.Option, 0, 4)

	if settings.Decompression {
		opts = append(opts, ginGzip.WithDecompressFn(decompressionFn))
	}

	if len(settings.Exclude.Extension) > 0 {
		opts = append(opts, ginGzip.WithExcludedExtensions(settings.Exclude.Extension))
	}

	if len(settings.Exclude.Path) > 0 {
		opts = append(opts, ginGzip.WithExcludedPaths(settings.Exclude.Path))
	}

	if len(settings.Exclude.PathRegex) > 0 {
		opts = append(opts, ginGzip.WithExcludedPathsRegexs(settings.Exclude.PathRegex))
	}

	middlewares = append(middlewares, ginGzip.Gzip(level, opts...))
	middlewares = append(middlewares, recordResponseSize)

	return middlewares, nil
}

func parseLevel(level string) (int, error) {
	switch level {
	case "none":
		return gzip.NoCompression, nil
	case "default":
		return gzip.DefaultCompression, nil
	case "best":
		return gzip.BestCompression, nil
	case "fast":
		return gzip.BestSpeed, nil
	default:
		if parsedLevel, err := strconv.ParseInt(level, 10, 64); err != nil {
			return 0, fmt.Errorf("failed to parse level %s: %w", level, err)
		} else if parsedLevel < gzip.NoCompression || parsedLevel > gzip.BestCompression {
			return 0, fmt.Errorf("invalid compression level %d", parsedLevel)
		} else {
			return int(parsedLevel), nil
		}
	}
}

func decompressionFn(c *gin.Context) {
	gzipReader, readUncompressedBytes, err := NewGZipBodyReader(c.Request.Body)
	if err != nil {
		// the body is not a proper gzip encoded body, so don't do anything
		// the client most likely set the wrong content encoding on the message
		return
	}

	c.Request.Body = gzipReader

	c.Set(requestCompressedSizeFields, encodedSizeData{
		sizeData: sizeData{
			size: readUncompressedBytes,
		},
		contentEncoding: "gzip",
	})
}

func recordRequestSize(c *gin.Context) {
	body, readBytes := NewCountingBodyReader(c.Request.Body)
	c.Request.Body = body

	c.Set(requestSizeFields, sizeData{
		size: readBytes,
	})

	c.Next()
}

func recordResponseSize(c *gin.Context) {
	if c.Writer.Header().Get("Content-Encoding") != "gzip" {
		c.Next()

		return
	}

	writer, writtenBytes := NewCountingBodyWriter(c.Writer)
	c.Writer = writer

	c.Set(responseSizeFields, encodedSizeData{
		sizeData: sizeData{
			size: writtenBytes,
		},
		contentEncoding: "gzip",
	})

	c.Next()
}
