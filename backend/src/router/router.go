// Package router 注册业务 HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/ctrl/question"
)

// RegisterRouter 注册题目模块路由，分组暂不添加中间件。
func RegisterRouter(router *gin.Engine) {
	questionGroup := router.Group("/api/v1/question")
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
