package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func registerUpstreamRoutes(apiRouter *gin.RouterGroup) {
	upstream := apiRouter.Group("/upstream")
	upstream.Use(middleware.AdminAuth())
	{
		upstream.GET("", controller.ListUpstreams)
		upstream.GET("/", controller.ListUpstreams)
		upstream.GET("/config", controller.GetUpstreamConfig)
		upstream.POST("/refresh", controller.RefreshAllUpstreams)
		upstream.GET("/:id/rankings", controller.GetUpstreamRankings)
		upstream.POST("/:id/refresh", controller.RefreshUpstream)
		upstream.GET("/:id", controller.GetUpstream)
		upstream.POST("", middleware.RootAuth(), controller.CreateUpstream)
		upstream.POST("/", middleware.RootAuth(), controller.CreateUpstream)
		upstream.PUT("/config", middleware.RootAuth(), controller.UpdateUpstreamConfig)
		upstream.PUT("/:id", middleware.RootAuth(), controller.UpdateUpstream)
		upstream.DELETE("/:id", middleware.RootAuth(), controller.DeleteUpstream)
	}
}
