package main

import (
	"log"

	sentinel "github.com/alibaba/sentinel-golang/api"
	sentinelConfig "github.com/alibaba/sentinel-golang/core/config"
	sentinelPlugin "github.com/alibaba/sentinel-golang/pkg/adapters/gin"
	"github.com/gin-gonic/gin"
	k8s "github.com/sentinel-group/sentinel-go-datasource-opensergo-poc"
)

func main() {
	err := sentinel.InitDefault()
	if err != nil {
		log.Fatal(err)
	}
	ds, err := k8s.NewDataSource("default", sentinelConfig.AppName())
	if err != nil {
		log.Fatal(err)
	}
	err = ds.RegisterControllersAndStart()
	if err != nil {
		log.Fatal(err)
	}

	// A Gin web example
	r := gin.New()
	r.Use(
		sentinelPlugin.SentinelMiddleware(
			// We'll support FallbackAction with OpenSergo spec in the future.
			sentinelPlugin.WithBlockFallback(func(ctx *gin.Context) {
				ctx.AbortWithStatusJSON(429, map[string]interface{}{
					"message": "Blocked by Sentinel",
				})
			}),
		),
	)

	// curl http://localhost:10000/foo
	r.GET("/foo", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	_ = r.Run(":10000")

	ch := make(chan struct{})
	<-ch
}
