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

type CommentUpdater interface {
	UpdateComment(context.Context, string, model.UpdateCommentReq) (model.UpdateCommentData, error)
}
type UpdateController struct{ service CommentUpdater }

func NewUpdateController(s CommentUpdater) *UpdateController { return &UpdateController{service: s} }
func UpdateComment(c *gin.Context) {
	NewUpdateController(service.NewUpdateCommentService(nil, tools.FilterSensitiveWords)).UpdateComment(c)
}

func (ctrl *UpdateController) UpdateComment(c *gin.Context) {
	userID := c.GetHeader("user_id")
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
		return
	}
	var req model.UpdateCommentReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.CommentID) == "" || strings.TrimSpace(req.Content) == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	result, err := ctrl.service.UpdateComment(c.Request.Context(), userID, req)
	switch {
	case errors.Is(err, service.ErrUnauthenticated):
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
	case errors.Is(err, service.ErrInvalidCommentInput):
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
	case errors.Is(err, service.ErrCommentForbidden):
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "无权限", "data": nil})
	case errors.Is(err, service.ErrCommentNotFound):
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "评论不存在", "data": nil})
	case err != nil:
		log.Printf("update comment: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
	default:
		c.JSON(http.StatusOK, model.UpdateCommentResp{Code: 0, Msg: "success", Data: result})
	}
}
