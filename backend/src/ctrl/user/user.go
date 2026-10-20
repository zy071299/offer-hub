// Package user 提供用户模块的路由入口。
package user

import (
	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/ctrl"
)

func GetUserInfo(c *gin.Context) { ctrl.GetUserInfo(c) }
