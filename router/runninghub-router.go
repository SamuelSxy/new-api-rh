package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetRunningHubRouter(router *gin.Engine) {
	rhRouter := router.Group("/runninghub/v1")
	rhRouter.Use(middleware.RouteTag("relay"))
	rhRouter.Use(middleware.TokenAuth(), middleware.Distribute())
	{
		// Image generation (imageGenerate)
		rhRouter.POST("/image", controller.RelayTask)
		// Video generation (auto-detect: image-to-video or text-to-video)
		rhRouter.POST("/video", controller.RelayTask)
		// Text output workflow (textOutput)
		rhRouter.POST("/text", controller.RelayTask)
		// Audio/TTS workflow (audioGenerate)
		rhRouter.POST("/audio", controller.RelayTask)
		// Task fetch (shared across all task types)
		rhRouter.GET("/task/:task_id", controller.RelayTaskFetch)
	}
}
