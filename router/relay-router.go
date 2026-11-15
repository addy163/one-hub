package router

import (
	"one-api/middleware"
	"one-api/relay"
	"one-api/relay/midjourney"
	"one-api/relay/task"
	"one-api/relay/task/kling"
	"one-api/relay/task/suno"

	"github.com/gin-gonic/gin"
)

func SetRelayRouter(router *gin.Engine) {
	router.Use(middleware.CORS())
	// https://platform.openai.com/docs/api-reference/introduction
	setOpenAIRouter(router)
	setNativeProtocolRouter(router)
	setTaskQueryRouter(router)
	setMJRouter(router)
	setSunoRouter(router)
	setClaudeRouter(router)
	setGeminiRouter(router)
	setRecraftRouter(router)
	setKlingRouter(router)
}

func setOpenAIRouter(router *gin.Engine) {
	modelsRouter := router.Group("/v1/models")
	modelsRouter.Use(middleware.UniversalAuth(), middleware.Distribute())
	{
		modelsRouter.GET("", relay.ListModelsAuto)
		modelsRouter.GET("/:model", relay.RetrieveModel)
	}
	relayV1Router := router.Group("/v1")
	relayV1Router.Use(middleware.RelayPanicRecover(), middleware.OpenaiAuth(), middleware.Distribute(), middleware.DynamicRedisRateLimiter())
	{
		relayV1Router.POST("/completions", relay.Relay)
		relayV1Router.POST("/chat/completions", relay.Relay)
		relayV1Router.POST("/responses", relay.Relay)
		// relayV1Router.POST("/edits", controller.Relay)
		relayV1Router.POST("/images/generations", relay.Relay)
		relayV1Router.POST("/images/edits", relay.Relay)
		relayV1Router.POST("/images/variations", relay.Relay)
		relayV1Router.POST("/embeddings", relay.Relay)
		// relayV1Router.POST("/engines/:model/embeddings", controller.RelayEmbeddings)
		relayV1Router.POST("/audio/transcriptions", relay.Relay)
		relayV1Router.POST("/audio/translations", relay.Relay)
		relayV1Router.POST("/audio/speech", relay.Relay)
		relayV1Router.POST("/moderations", relay.Relay)
		relayV1Router.POST("/rerank", relay.RelayRerank)
		relayV1Router.GET("/realtime", relay.ChatRealtime)

		relayV1Router.Use(middleware.SpecifiedChannel())
		{
			relayV1Router.Any("/files", relay.RelayOnly)
			relayV1Router.Any("/files/*any", relay.RelayOnly)
			relayV1Router.Any("/fine_tuning/*any", relay.RelayOnly)
			relayV1Router.Any("/assistants", relay.RelayOnly)
			relayV1Router.Any("/assistants/*any", relay.RelayOnly)
			relayV1Router.Any("/threads", relay.RelayOnly)
			relayV1Router.Any("/threads/*any", relay.RelayOnly)
			relayV1Router.Any("/batches/*any", relay.RelayOnly)
			relayV1Router.Any("/vector_stores/*any", relay.RelayOnly)
			relayV1Router.DELETE("/models/:model", relay.RelayOnly)

			// Responses API 的配套端点：retrieve / delete / cancel / input_items /
			// compact / background 结果读取等，以及会话端点。这些请求没有 model 字段，
			// 无法按模型选渠道，沿用上面 files/assistants 的约定（令牌带 #渠道ID），
			// 路径、method 与 body 原样透传上游，不经过任何协议转换。
			relayV1Router.Any("/responses/*any", relay.RelayOnly)
			relayV1Router.Any("/conversations", relay.RelayOnly)
			relayV1Router.Any("/conversations/*any", relay.RelayOnly)
		}
	}
}

// setNativeProtocolRouter 为 Anthropic / Gemini 原生 SDK 提供标准路径别名。
// 原有 /claude/v1/messages 与 /gemini/:version/models/:model 要求客户端额外配置
// URL 前缀，Claude Code、Gemini SDK 开箱即用会失败。这里复用完全相同的中继与
// 鉴权链路，只补一层标准路径。
func setNativeProtocolRouter(router *gin.Engine) {
	// 路由始终注册，由 NativeProtocolGate 在请求时读取开关，
	// 后台修改 passthrough_native_protocol 后即可立即生效。
	claudeNative := router.Group("/v1")
	claudeNative.Use(middleware.NativeProtocolGate(), middleware.RelayCluadePanicRecover(), middleware.ClaudeAuth(), middleware.Distribute(), middleware.DynamicRedisRateLimiter())
	{
		claudeNative.POST("/messages", relay.Relay)
	}

	geminiNative := router.Group("/v1beta")
	geminiNative.Use(middleware.NativeProtocolGate(), middleware.RelayGeminiPanicRecover(), middleware.GeminiAuth(), middleware.Distribute(), middleware.DynamicRedisRateLimiter())
	{
		geminiNative.GET("/models", relay.ListGeminiModelsByToken)
		geminiNative.GET("/openai/models", relay.ListModelsByToken)
		geminiNative.POST("/models/:model", relay.Relay)
	}
}

// setTaskQueryRouter 提供与平台无关的只读任务状态查询，
// 复用现有 Task 表，不改动 MJ / Suno / Kling 各自的私有查询路径。
func setTaskQueryRouter(router *gin.Engine) {
	tasks := router.Group("/v1/tasks")
	tasks.Use(middleware.UniversalAuth())
	{
		tasks.GET("/:key", relay.GetTaskByKey)
		tasks.GET("/:key/artifacts", relay.GetTaskArtifacts)
	}

	videos := router.Group("/v1/videos")
	videos.Use(middleware.UniversalAuth())
	{
		// 本机任务表直出，供 one-hub 自己的 Suno / Kling / MJ 任务使用。
		videos.GET("/:task_id", relay.GetVideoTaskByID)
		// 下列端点交给上游处理：提交、remix、内容下载、删除。
		// 没有 model 字段，因此要求令牌带 #渠道ID。
		videos.POST("", middleware.SpecifiedChannel(), relay.RelayOnly)
		videos.POST("/:task_id/remix", middleware.SpecifiedChannel(), relay.RelayOnly)
		videos.GET("/:task_id/content", middleware.SpecifiedChannel(), relay.RelayOnly)
		videos.DELETE("/:task_id", middleware.SpecifiedChannel(), relay.RelayOnly)
	}
}

func setMJRouter(router *gin.Engine) {
	relayMjRouter := router.Group("/mj")
	registerMjRouterGroup(relayMjRouter)

	relayMjModeRouter := router.Group("/:mode/mj")
	registerMjRouterGroup(relayMjModeRouter)
}

// Author: Calcium-Ion
// GitHub: https://github.com/Calcium-Ion/new-api
// Path: router/relay-router.go
func registerMjRouterGroup(relayMjRouter *gin.RouterGroup) {
	relayMjRouter.GET("/image/:id", midjourney.RelayMidjourneyImage)
	relayMjRouter.Use(middleware.RelayMJPanicRecover(), middleware.MjAuth(), middleware.Distribute(), middleware.DynamicRedisRateLimiter())
	{
		relayMjRouter.POST("/submit/action", midjourney.RelayMidjourney)
		relayMjRouter.POST("/submit/shorten", midjourney.RelayMidjourney)
		relayMjRouter.POST("/submit/modal", midjourney.RelayMidjourney)
		relayMjRouter.POST("/submit/imagine", midjourney.RelayMidjourney)
		relayMjRouter.POST("/submit/change", midjourney.RelayMidjourney)
		relayMjRouter.POST("/submit/simple-change", midjourney.RelayMidjourney)
		relayMjRouter.POST("/submit/describe", midjourney.RelayMidjourney)
		relayMjRouter.POST("/submit/blend", midjourney.RelayMidjourney)
		relayMjRouter.POST("/notify", midjourney.RelayMidjourney)
		relayMjRouter.GET("/task/:id/fetch", midjourney.RelayMidjourney)
		relayMjRouter.GET("/task/:id/image-seed", midjourney.RelayMidjourney)
		relayMjRouter.POST("/task/list-by-condition", midjourney.RelayMidjourney)
		relayMjRouter.POST("/insight-face/swap", midjourney.RelayMidjourney)
		relayMjRouter.POST("/submit/upload-discord-images", midjourney.RelayMidjourney)
	}
}

func setSunoRouter(router *gin.Engine) {
	relaySunoRouter := router.Group("/suno")
	relaySunoRouter.Use(middleware.RelaySunoPanicRecover(), middleware.OpenaiAuth(), middleware.Distribute(), middleware.DynamicRedisRateLimiter())
	{
		relaySunoRouter.POST("/submit/:action", task.RelayTaskSubmit)
		relaySunoRouter.POST("/fetch", suno.GetFetch)
		relaySunoRouter.GET("/fetch/:id", suno.GetFetchByID)
	}
}

func setClaudeRouter(router *gin.Engine) {
	relayClaudeRouter := router.Group("/claude")
	relayV1Router := relayClaudeRouter.Group("/v1")
	relayV1Router.Use(middleware.APIEnabled("claude"), middleware.RelayCluadePanicRecover(), middleware.ClaudeAuth(), middleware.Distribute(), middleware.DynamicRedisRateLimiter())
	{
		relayV1Router.POST("/messages", relay.Relay)
		relayV1Router.GET("/models", relay.ListClaudeModelsByToken)
	}
}

func setGeminiRouter(router *gin.Engine) {
	relayGeminiRouter := router.Group("/gemini")
	relayGeminiRouter.Use(middleware.APIEnabled("gemini"), middleware.RelayGeminiPanicRecover(), middleware.GeminiAuth(), middleware.Distribute(), middleware.DynamicRedisRateLimiter())
	{
		relayGeminiRouter.POST("/:version/models/:model", relay.Relay)
		relayGeminiRouter.GET("/:version/models", relay.ListGeminiModelsByToken)
	}
}

func setRecraftRouter(router *gin.Engine) {
	relayRecraftRouter := router.Group("/recraftAI/v1")
	relayRecraftRouter.Use(middleware.RelayPanicRecover(), middleware.OpenaiAuth(), middleware.Distribute(), middleware.DynamicRedisRateLimiter())
	{
		relayRecraftRouter.POST("/images/generations", relay.Relay)
		relayRecraftRouter.POST("/images/vectorize", relay.RelayRecraftAI)
		relayRecraftRouter.POST("/images/removeBackground", relay.RelayRecraftAI)
		relayRecraftRouter.POST("/images/clarityUpscale", relay.RelayRecraftAI)
		relayRecraftRouter.POST("/images/generativeUpscale", relay.RelayRecraftAI)
		relayRecraftRouter.POST("/styles", relay.RelayRecraftAI)
	}
}

func setKlingRouter(router *gin.Engine) {
	relayKlingRouter := router.Group("/kling")
	relayKlingRouter.Use(middleware.RelayKlingPanicRecover(), middleware.OpenaiAuth(), middleware.Distribute())
	relayKlingRouter.GET("/v1/videos/text2video/:id", kling.GetFetchByID)
	relayKlingRouter.GET("/v1/videos/image2video/:id", kling.GetFetchByID)

	relayKlingRouter.Use(middleware.DynamicRedisRateLimiter())
	{
		relayKlingRouter.POST("/v1/:class/:action", task.RelayTaskSubmit)
	}
}
