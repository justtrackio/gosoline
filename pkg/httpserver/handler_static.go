package httpserver

import (
	"embed"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func CreateStaticHandler(files embed.FS, dir string, excludes ...string) (func(c *gin.Context), error) {
	dist, err := fs.Sub(files, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to sub %q directory: %w", dir, err)
	}

	return func(c *gin.Context) {
		for _, exclude := range excludes {
			if strings.HasPrefix(c.Request.URL.Path, exclude) {
				return
			}
		}

		path := c.Request.URL.Path

		if path == "/" {
			path = "/index.html"
		}

		path = strings.TrimPrefix(path, "/")
		file, err := dist.Open(path)
		if err != nil {
			c.String(http.StatusNotFound, "failed to open %s: %w", path, err)
			c.Abort()

			return
		}

		contentType := mime.TypeByExtension(filepath.Ext(path))
		stat, err := file.Stat()
		if err != nil {
			c.String(http.StatusNotFound, "failed to get file size %s: %w", path, err)
			c.Abort()

			return
		}

		c.DataFromReader(http.StatusOK, stat.Size(), contentType, file, map[string]string{})
		c.Abort()
	}, nil
}
