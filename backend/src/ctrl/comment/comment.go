package comment

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/model"
	"offer-hub/backend/src/service"
)

type CommentLister interface {
	ListComments(context.Context, model.ListCommentsReq) (model.ListCommentsData, error)
}
type Controller struct{ service CommentLister }

func NewController(s CommentLister) *Controller { return &Controller{service: s} }
func ListComments(c *gin.Context)               { NewController(service.NewCommentService(nil, nil)).ListComments(c) }
func (ctrl *Controller) ListComments(c *gin.Context) {
	req := model.ListCommentsReq{Page: 1, PageSize: 20, SubCommentPage: 1, SubCommentSize: 5, SortBy: "create_time", SortOrder: "desc"}
	if _, err := url.ParseQuery(c.Request.URL.RawQuery); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	if err := service.ValidateCommentQuery(req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	req.UserID = c.GetHeader("user_id")
	result, err := ctrl.service.ListComments(c.Request.Context(), req)
	if errors.Is(err, service.ErrInvalidCommentQuery) {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	if err != nil {
		log.Printf("list comments: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
		return
	}
	c.JSON(http.StatusOK, model.ListCommentsResp{Code: 0, Msg: "success", Data: result})
}
