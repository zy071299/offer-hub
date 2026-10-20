// Package router 注册业务 HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/ctrl/auth"
	"offer-hub/backend/src/ctrl/comment"
	"offer-hub/backend/src/ctrl/question"
	"offer-hub/backend/src/ctrl/tools"
	"offer-hub/backend/src/ctrl/user"
)

// RegisterRouter 公开评论无需鉴权，题目组使用软鉴权，评论写操作、登出及用户信息使用强制鉴权。
func RegisterRouter(router *gin.Engine) {
	router.POST("/auth/register", auth.Register)
	router.POST("/auth/login", auth.Login)
	openGroup := router.Group("/api/v1/open")
	{
		openGroup.GET("/list_comments", comment.ListComments)
	}
	router.POST("/auth/logout", tools.JWTAuthMiddleware(), auth.Logout)
	commentGroup := router.Group("/api/v1/comment", tools.JWTAuthMiddleware())
	{
		commentGroup.POST("/add", comment.AddComment)
		commentGroup.POST("/delete", comment.DeleteComment)
		commentGroup.POST("/update", comment.UpdateComment)
	}
	userGroup := router.Group("/api/v1/user_info", tools.JWTAuthMiddleware())
	{
		userGroup.GET("/get_user_info", user.GetUserInfo)
	}
	questionGroup := router.Group("/api/v1/question", tools.SoftJWTAuthMiddleware())
	{
		questionGroup.GET("/all/list", question.GetQuestionBankSeries)
		questionGroup.GET("/list", question.ListQuestions)
		questionGroup.GET("/meta/list", question.ListQuestionMeta)
		questionGroup.GET("/detail", question.GetQuestionDetail)
		questionGroup.GET("/hot/list", question.GetHotQuestionList)
	}
}

// New 保留现有创建接口；启动入口显式分开创建引擎与注册路由。
func New() *gin.Engine {
	router := gin.Default()
	RegisterRouter(router)
	return router
}
