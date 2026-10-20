// Package auth 提供鉴权模块的路由入口。
package auth

import (
	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/ctrl"
)

func Register(c *gin.Context) { ctrl.PasswordRegister(c) }

func Login(c *gin.Context) { ctrl.PasswordLogin(c) }

func Logout(c *gin.Context) { ctrl.Logout(c) }
