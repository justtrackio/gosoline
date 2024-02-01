package httpserver

import (
	"net/http"
	"net/http/pprof"

	"github.com/gin-gonic/gin"
)

const (
	BaseProfiling = "/debug/profiling"
	CmdLine       = "/cmdline"
	Profile       = "/profile"
	Symbol        = "/symbol"
	Trace         = "/trace"
	Allocs        = "/allocs"
	Block         = "/block"
	GoRoutine     = "/goroutine"
	Heap          = "/heap"
	Mutex         = "/mutex"
	ThreadCreate  = "/threadcreate"
)

func AddProfilingEndpoints(r *gin.Engine) {
	pr := r.Group(BaseProfiling)
	pr.GET("/", profilingHandler(pprof.Index))
	pr.GET(CmdLine, profilingHandler(pprof.Cmdline))
	pr.GET(Profile, profilingHandler(pprof.Profile))
	pr.POST(Symbol, profilingHandler(pprof.Symbol))
	pr.GET(Symbol, profilingHandler(pprof.Symbol))
	pr.GET(Trace, profilingHandler(pprof.Trace))
	pr.GET(Allocs, profilingHandler(pprof.Handler("allocs").ServeHTTP))
	pr.GET(Block, profilingHandler(pprof.Handler("block").ServeHTTP))
	pr.GET(GoRoutine, profilingHandler(pprof.Handler("goroutine").ServeHTTP))
	pr.GET(Heap, profilingHandler(pprof.Handler("heap").ServeHTTP))
	pr.GET(Mutex, profilingHandler(pprof.Handler("mutex").ServeHTTP))
	pr.GET(ThreadCreate, profilingHandler(pprof.Handler("threadcreate").ServeHTTP))
}

func profilingHandler(handler http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
