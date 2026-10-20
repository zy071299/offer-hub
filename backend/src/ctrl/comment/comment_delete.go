package comment

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

type CommentDeleter interface {
	DeleteComment(context.Context, string, string) error
}
type DeleteController struct{ service CommentDeleter }

func NewDeleteController(s CommentDeleter) *DeleteController { return &DeleteController{service: s} }
func DeleteComment(c *gin.Context) {
	NewDeleteController(service.NewDeleteCommentService(nil)).DeleteComment(c)
}

func (ctrl *DeleteController) DeleteComment(c *gin.Context) {
	userID := c.GetHeader("user_id")
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
		return
	}
	var req model.DeleteCommentReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.CommentID) == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	err := ctrl.service.DeleteComment(c.Request.Context(), userID, req.CommentID)
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
		log.Printf("delete comment: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
	default:
		c.JSON(http.StatusOK, model.DeleteCommentResp{Code: 0, Msg: "success"})
	}
}
