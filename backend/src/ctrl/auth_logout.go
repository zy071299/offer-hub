package ctrl

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/config"
	"offer-hub/backend/src/service"
)

type TokenLogoutService interface {
	Logout(context.Context, string) error
}

type LogoutController struct{ service TokenLogoutService }

func NewLogoutController(logoutService TokenLogoutService) *LogoutController {
	return &LogoutController{service: logoutService}
}

func Logout(c *gin.Context) {
	NewLogoutController(service.NewLogoutService(nil, config.Conf.JWT)).Logout(c)
}

func (ctrl *LogoutController) Logout(c *gin.Context) {
	headers := c.Request.Header.Values("Authorization")
	if len(headers) != 1 {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
		return
	}
	parts := strings.Fields(headers[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
		return
	}
	err := ctrl.service.Logout(c.Request.Context(), parts[1])
	switch {
	case errors.Is(err, service.ErrUnauthenticated):
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
	case err != nil:
		log.Printf("logout failed: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
	default:
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": gin.H{"message": "登出成功"}})
	}
}
