package apiserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Definer func(ctx context.Context, config cfg.Config, logger log.Logger) (*Definitions, error)

type Definition struct {
	group        *Definitions
	httpMethod   string
	relativePath string
	handlers     []gin.HandlerFunc
}

func (d *Definition) getAbsolutePath() string {
	groupPath := d.group.getAbsolutePath()

	absolutePath := fmt.Sprintf("%s/%s", groupPath, d.relativePath)
	absolutePath = strings.TrimRight(absolutePath, "/")

	return removeDuplicates(absolutePath)
}

type Definitions struct {
	basePath   string
	middleware []gin.HandlerFunc
	routes     []Definition

	children []*Definitions
	parent   *Definitions
}

func (d *Definitions) getAbsolutePath() string {
	parentPath := "/"

	if d.parent != nil {
		parentPath = d.parent.getAbsolutePath()
	}

	absolutePath := fmt.Sprintf("%s/%s", parentPath, d.basePath)

	return removeDuplicates(absolutePath)
}

func (d *Definitions) Group(relativePath string) *Definitions {
	newGroup := &Definitions{
		basePath: relativePath,
		children: make([]*Definitions, 0),
		parent:   d,
	}

	d.children = append(d.children, newGroup)

	return newGroup
}

func (d *Definitions) Use(middleware ...gin.HandlerFunc) {
	d.middleware = append(d.middleware, middleware...)
}

func (d *Definitions) Handle(httpMethod, relativePath string, handlers ...gin.HandlerFunc) {
	relativePath = strings.TrimRight(relativePath, "/")

	d.routes = append(d.routes, Definition{
		group:        d,
		httpMethod:   httpMethod,
		relativePath: relativePath,
		handlers:     handlers,
	})
}

func (d *Definitions) POST(relativePath string, handlers ...gin.HandlerFunc) {
	d.Handle(http.PostRequest, relativePath, handlers...)
}

func (d *Definitions) GET(relativePath string, handlers ...gin.HandlerFunc) {
	d.Handle(http.GetRequest, relativePath, handlers...)
}

func (d *Definitions) DELETE(relativePath string, handlers ...gin.HandlerFunc) {
	d.Handle(http.DeleteRequest, relativePath, handlers...)
}

func (d *Definitions) PUT(relativePath string, handlers ...gin.HandlerFunc) {
	d.Handle(http.PutRequest, relativePath, handlers...)
}

func (d *Definitions) OPTIONS(relativePath string, handlers ...gin.HandlerFunc) {
	d.Handle(http.OptionsRequest, relativePath, handlers...)
}

func buildRouter(definitions *Definitions, router gin.IRouter) []Definition {
	var definitionList []Definition
	grp := router

	if definitions.parent != nil {
		grp = router.Group(definitions.basePath)
	}

	for _, m := range definitions.middleware {
		grp.Use(m)
	}

	for _, d := range definitions.routes {
		handlers := make([]gin.HandlerFunc, 0, len(d.handlers)+1)
		handlers = append(handlers, d.handlers...)

		grp.Handle(d.httpMethod, d.relativePath, handlers...)
	}

	definitionList = append(definitionList, definitions.routes...)

	for _, c := range definitions.children {
		definitionList = append(definitionList, buildRouter(c, grp)...)
	}

	return definitionList
}

func removeDuplicates(s string) string {
	var buf strings.Builder
	var last rune

	for i, r := range s {
		if i == 0 || r != '/' || r != last {
			buf.WriteRune(r)
		}

		last = r
	}

	return buf.String()
}
