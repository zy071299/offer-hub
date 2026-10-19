// Package question 提供题目 HTTP 控制器。
package question

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/model"
	"offer-hub/backend/src/service"
)

type QuestionLister interface {
	GetAllQuestionList(context.Context, model.GetAllQuestionListReq) ([]model.GetAllQuestionListData, error)
	ListQuestions(context.Context, model.ListQuestionReq) (model.ListQuestionData, error)
	ListQuestionsMeta(context.Context, model.ListQuestionMetaReq) (model.ListQuestionMetaResp, error)
	GetQuestionDetail(context.Context, model.GetQuestionDetailReq) (model.OneQuestion, error)
	GetHotQuestions(context.Context, model.GetHotQuestionsReq) (model.GetHotQuestionsResp, error)
}

type Controller struct{ service QuestionLister }

func NewController(questionService QuestionLister) *Controller {
	return &Controller{service: questionService}
}

// GetQuestionBankSeries 使用默认业务服务，游客和登录用户均可访问。
func GetQuestionBankSeries(c *gin.Context) {
	NewController(service.NewQuestionService(nil)).GetAllQuestionList(c)
}

func (ctrl *Controller) GetAllQuestionList(c *gin.Context) {
	var req model.GetAllQuestionListReq
	// 显式校验原始 query 的语法，避免 URL.Query 静默丢弃非法参数。
	if _, err := url.ParseQuery(c.Request.URL.RawQuery); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	result, err := ctrl.service.GetAllQuestionList(c.Request.Context(), req)
	if err != nil {
		log.Printf("get all question list: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
		return
	}
	c.JSON(http.StatusOK, model.GetAllQuestionListResp{Code: 0, Msg: "success", Data: result})
}

// ListQuestions 提供默认游客题目列表处理器。
func ListQuestions(c *gin.Context) {
	NewController(service.NewQuestionService(nil)).ListQuestions(c)
}

func (ctrl *Controller) ListQuestions(c *gin.Context) {
	req, err := bindListQuestionQuery(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	result, err := ctrl.service.ListQuestions(c.Request.Context(), req)
	if err != nil {
		log.Printf("list questions: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
		return
	}
	c.JSON(http.StatusOK, model.ListQuestionResp{Code: 0, Msg: "success", Data: result})
}

func ListQuestionMeta(c *gin.Context) {
	NewController(service.NewQuestionService(nil)).ListQuestionsMeta(c)
}

func (ctrl *Controller) ListQuestionsMeta(c *gin.Context) {
	req, err := bindListQuestionQuery(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	result, err := ctrl.service.ListQuestionsMeta(c.Request.Context(), req)
	if err != nil {
		log.Printf("list questions meta: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
		return
	}
	c.JSON(http.StatusOK, result)
}

// bindListQuestionQuery 保证题目列表与元数据列表具有相同参数、默认值和校验。
func bindListQuestionQuery(c *gin.Context) (model.ListQuestionReq, error) {
	req := model.ListQuestionReq{Page: 1, PageSize: 20, SortBy: "create_time", SortOrder: "desc"}
	if _, err := url.ParseQuery(c.Request.URL.RawQuery); err != nil {
		return req, err
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		return req, err
	}
	if req.SortBy == "" {
		req.SortBy = "create_time"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}
	if req.Page < 1 || req.PageSize < 1 || int64(req.Page-1) > (1<<63-1)/int64(req.PageSize) || req.Difficulty < 0 || req.Difficulty > 3 {
		return req, errors.New("invalid pagination or difficulty")
	}
	switch req.SortBy {
	case "create_time", "view_count", "thumbs_up_count", "dislike_count":
	default:
		return req, errors.New("invalid sort field")
	}
	if req.SortOrder != "asc" && req.SortOrder != "desc" {
		return req, errors.New("invalid sort order")
	}
	return req, nil
}

func GetQuestionDetail(c *gin.Context) {
	NewController(service.NewQuestionService(nil)).GetQuestionDetail(c)
}

func (ctrl *Controller) GetQuestionDetail(c *gin.Context) {
	var req model.GetQuestionDetailReq
	if _, err := url.ParseQuery(c.Request.URL.RawQuery); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	if err := c.ShouldBindQuery(&req); err != nil || strings.TrimSpace(req.QuestionID) == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "question_id 必填", "data": nil})
		return
	}
	result, err := ctrl.service.GetQuestionDetail(c.Request.Context(), req)
	if errors.Is(err, service.ErrQuestionNotFound) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "题目不存在", "data": nil})
		return
	}
	if err != nil {
		log.Printf("get question detail: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
		return
	}
	c.JSON(http.StatusOK, model.GetQuestionDetailResp{Code: 0, Msg: "success", Data: result})
}

func GetHotQuestionList(c *gin.Context) {
	NewController(service.NewQuestionService(nil)).GetHotQuestions(c)
}

func (ctrl *Controller) GetHotQuestions(c *gin.Context) {
	req := model.GetHotQuestionsReq{Limit: 10}
	if _, err := url.ParseQuery(c.Request.URL.RawQuery); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误", "data": nil})
		return
	}
	if err := c.ShouldBindQuery(&req); err != nil || req.Limit < 1 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "limit 必须为正整数", "data": nil})
		return
	}
	result, err := ctrl.service.GetHotQuestions(c.Request.Context(), req)
	if err != nil {
		log.Printf("get hot questions: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
		return
	}
	c.JSON(http.StatusOK, result)
}
