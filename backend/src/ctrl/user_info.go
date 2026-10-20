package ctrl

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/model"
	"offer-hub/backend/src/service"
)

type UserInfoGetter interface {
	GetUserInfo(context.Context, string) (model.UserInfoData, error)
}

type UserInfoController struct{ service UserInfoGetter }

func NewUserInfoController(userService UserInfoGetter) *UserInfoController {
	return &UserInfoController{service: userService}
}

func GetUserInfo(c *gin.Context) {
	NewUserInfoController(service.NewUserInfoService(nil)).GetUserInfo(c)
}

func (ctrl *UserInfoController) GetUserInfo(c *gin.Context) {
	// 当前阶段按约定读取测试 Header；JWT 中间件接入后切换为可信身份。
	identities := c.Request.Header.Values("user_id")
	if len(identities) != 1 || strings.TrimSpace(identities[0]) == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
		return
	}
	result, err := ctrl.service.GetUserInfo(c.Request.Context(), strings.TrimSpace(identities[0]))
	switch {
	case errors.Is(err, service.ErrUnauthenticated):
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
	case errors.Is(err, service.ErrUserInfoNotFound):
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "用户不存在", "data": nil})
	case err != nil:
		log.Printf("get user info: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
	default:
		c.JSON(http.StatusOK, model.UserInfoResp{Code: 0, Msg: "success", Data: result})
	}
}
