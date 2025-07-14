package httpserver

import (
	"regexp"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

func Cors(config cfg.Config) (gin.HandlerFunc, error) {
	allowedOriginPattern, err := config.GetString("api_cors_allowed_origin_pattern")
	if err != nil {
		return nil, err
	}
	validOrigin := regexp.MustCompile(allowedOriginPattern)

	allowedHeaders, err := config.GetStringSlice("api_cors_allowed_headers")
	if err != nil {
		return nil, err
	}
	allowedMethods, err := config.GetStringSlice("api_cors_allowed_methods")
	if err != nil {
		return nil, err
	}

	return cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return validOrigin.MatchString(origin)
		},
		AllowHeaders:     allowedHeaders,
		AllowMethods:     allowedMethods,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}), nil
}
