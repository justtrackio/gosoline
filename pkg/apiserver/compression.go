package apiserver

import (
	compressGzip "compress/gzip"
	"fmt"
	ginGzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"io"
	"strconv"
)

// CompressionSettings allow the enabling of gzip support for requests and responses. By default compressed requests are accepted, and compressed responses are returned (if suitable).
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

func configureCompression(router *gin.Engine, settings CompressionSettings) error {
	level, err := parseLevel(settings.Level)
	if err != nil {
		return err
	}

	if level == compressGzip.NoCompression && !settings.Decompression {
		// there is no use in adding a handler if we should neither compress nor decompress
		return nil
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

	router.Use(ginGzip.Gzip(level, opts...))

	return nil
}

func parseLevel(level string) (int, error) {
	switch level {
	case "none":
		return compressGzip.NoCompression, nil
	case "default":
		return compressGzip.DefaultCompression, nil
	case "best":
		return compressGzip.BestCompression, nil
	case "fast":
		return compressGzip.BestSpeed, nil
	default:
		if i, err := strconv.ParseInt(level, 10, 64); err != nil {
			return 0, fmt.Errorf("failed to parse level %s: %w", level, err)
		} else {
			return int(i), nil
		}
	}
}

type gzipBodyReader struct {
	body   io.ReadCloser
	reader *compressGzip.Reader
}

func (r gzipBodyReader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r gzipBodyReader) Close() error {
	err := r.reader.Close()
	if err != nil {
		return err
	}

	return r.body.Close()
}

func decompressionFn(c *gin.Context) {
	reader, err := compressGzip.NewReader(c.Request.Body)

	if err != nil {
		// the body is not a proper gzip encoded body, so don't do anything
		// the client most likely set the wrong content encoding on the message
		return
	}

	c.Request.Body = gzipBodyReader{
		body:   c.Request.Body,
		reader: reader,
	}
}
