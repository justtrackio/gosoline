package stream

import (
	"github.com/gin-gonic/gin"
)

type Definitions struct {
	basePath   string
	middleware []gin.HandlerFunc

	children []*Definitions
	parent   *Definitions
}
