package ctrl

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/model"
	"offer-hub/backend/src/service"
)

type PasswordRegistrar interface {
	PasswordRegister(context.Context, model.PasswordRegisterReq) error
}

type AuthController struct{ service PasswordRegistrar }

func NewAuthController(authService PasswordRegistrar) *AuthController {
	return &AuthController{service: authService}
}

func PasswordRegister(c *gin.Context) {
	NewAuthController(service.NewAuthService(nil)).PasswordRegister(c)
}

func (ctrl *AuthController) PasswordRegister(c *gin.Context) {
	var req model.PasswordRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "用户名或密码不符合要求", "data": nil})
		return
	}
	err := ctrl.service.PasswordRegister(c.Request.Context(), req)
	switch {
	case errors.Is(err, service.ErrInvalidRegister):
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "用户名或密码不符合要求", "data": nil})
	case errors.Is(err, service.ErrUsernameExists):
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "用户名已存在", "data": nil})
	case err != nil:
		log.Printf("password register: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
	default:
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "注册成功", "data": nil})
	}
}
