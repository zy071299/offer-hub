package ctrl

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/config"
	"offer-hub/backend/src/model"
	"offer-hub/backend/src/service"
)

type PasswordLoginService interface {
	PasswordLogin(context.Context, model.PasswordLoginReq) (model.PasswordLoginData, error)
}

type LoginController struct{ service PasswordLoginService }

func NewLoginController(loginService PasswordLoginService) *LoginController {
	return &LoginController{service: loginService}
}

func PasswordLogin(c *gin.Context) {
	NewLoginController(service.NewLoginService(nil, config.Conf.JWT)).PasswordLogin(c)
}

func (ctrl *LoginController) PasswordLogin(c *gin.Context) {
	var req model.PasswordLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "用户名或密码不符合要求", "data": nil})
		return
	}
	result, err := ctrl.service.PasswordLogin(c.Request.Context(), req)
	switch {
	case errors.Is(err, service.ErrInvalidLogin):
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "用户名或密码不符合要求", "data": nil})
	case errors.Is(err, service.ErrInvalidCredentials):
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "用户名或密码错误", "data": nil})
	case err != nil:
		log.Printf("password login: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
	default:
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "登录成功", "data": result})
	}
}
