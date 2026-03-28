package x

import (
	"msgPushSite/internal/context"
	"msgPushSite/mdata"

	"github.com/gin-gonic/gin"
)

type Handler func(c *context.Context)

func convert(h Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		c1 := &context.Context{Context: c, Trace: c.GetHeader(mdata.HeaderTrace)}
		h(c1)
	}
}

func POST(group *gin.RouterGroup, relativePath string, handler Handler) {
	group.POST(relativePath, convert(handler))
}

func GET(group *gin.RouterGroup, relativePath string, handler Handler) {
	group.GET(relativePath, convert(handler))
}

func PUT(group *gin.RouterGroup, relativePath string, handler Handler) {
	group.PUT(relativePath, convert(handler))
}

func DELETE(group *gin.RouterGroup, relativePath string, handler Handler) {
	group.DELETE(relativePath, convert(handler))
}

func Any(group *gin.RouterGroup, relativePath string, handler Handler) {
	group.Any(relativePath, convert(handler))
}
