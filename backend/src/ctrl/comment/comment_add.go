package comment

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/ctrl/tools"
	"offer-hub/backend/src/model"
	"offer-hub/backend/src/service"
)

type CommentAdder interface {
	AddComment(context.Context, string, model.AddCommentReq) (model.AddCommentData, error)
}
type AddController struct{ service CommentAdder }

func NewAddController(s CommentAdder) *AddController { return &AddController{service: s} }
func AddComment(c *gin.Context) {
	NewAddController(service.NewAddCommentService(nil, nil, tools.FilterSensitiveWords)).AddComment(c)
}
func (ctrl *AddController) AddComment(c *gin.Context) {
	userID := c.GetHeader("user_id")
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
		return
	}
	var req model.AddCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	result, err := ctrl.service.AddComment(c.Request.Context(), userID, req)
	switch {
	case errors.Is(err, service.ErrCommentCompensationFailed):
		log.Printf("add comment compensation failed: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
	case errors.Is(err, service.ErrInvalidCommentInput):
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
	case errors.Is(err, service.ErrUnauthenticated):
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
	case errors.Is(err, service.ErrCommentParentNotFound):
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "父评论不存在", "data": nil})
	case err != nil:
		log.Printf("add comment: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
	default:
		c.JSON(http.StatusOK, model.AddCommentResp{Code: 0, Msg: "success", Data: result})
	}
}
