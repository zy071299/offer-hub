// Package router 注册业务 HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"
)

// RegisterRouter 将业务路由绑定到已创建的引擎。
// 当前没有业务接口；后续在此绑定 ctrl 层处理函数。
func RegisterRouter(router *gin.Engine) {
}

// New 保留现有创建接口；启动入口显式分开创建引擎与注册路由。
func New() *gin.Engine {
	router := gin.Default()
	RegisterRouter(router)
	return router
}
