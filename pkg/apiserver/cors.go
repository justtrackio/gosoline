package apiserver

import (
	"regexp"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

func Cors(config cfg.Config) gin.HandlerFunc {
	allowedOriginPattern := config.GetString("api_cors_allowed_origin_pattern")
	validOrigin := regexp.MustCompile(allowedOriginPattern)

	allowedHeaders := config.GetStringSlice("api_cors_allowed_headers")
	allowedMethods := config.GetStringSlice("api_cors_allowed_methods")

	return cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return validOrigin.MatchString(origin)
		},
		AllowHeaders:     allowedHeaders,
		AllowMethods:     allowedMethods,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
