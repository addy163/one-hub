package controller

import (
	"errors"
	"net/http"
	"strings"

	"one-api/common"
	"one-api/model"

	"github.com/gin-gonic/gin"
)

// noticeRequest 后台提交的公告内容。
type noticeRequest struct {
	Id         int    `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Level      string `json:"level"`
	Pinned     bool   `json:"pinned"`
	ShowBanner bool   `json:"show_banner"`
	Popup      bool   `json:"popup"`
	RequireAck bool   `json:"require_ack"`
	Status     int    `json:"status"`
	StartTime  int64  `json:"start_time"`
	EndTime    int64  `json:"end_time"`
}

func (r *noticeRequest) validate() error {
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" {
		return errors.New("公告标题不能为空")
	}
	if len([]rune(r.Title)) > 200 {
		return errors.New("公告标题不能超过 200 个字符")
	}
	switch r.Level {
	case model.NoticeLevelInfo, model.NoticeLevelSuccess, model.NoticeLevelWarning, model.NoticeLevelError, "":
	default:
		return errors.New("公告级别无效")
	}
	if r.Level == "" {
		r.Level = model.NoticeLevelInfo
	}
	switch r.Status {
	case model.NoticeStatusDraft, model.NoticeStatusOnline, model.NoticeStatusOffline:
	default:
		return errors.New("公告状态无效")
	}
	if r.StartTime < 0 || r.EndTime < 0 {
		return errors.New("生效时间不能为负数")
	}
	if r.StartTime > 0 && r.EndTime > 0 && r.EndTime <= r.StartTime {
		return errors.New("结束时间必须晚于开始时间")
	}
	return nil
}

func (r *noticeRequest) toModel() *model.Notice {
	return &model.Notice{
		Id:         r.Id,
		Title:      r.Title,
		Content:    r.Content,
		Level:      r.Level,
		Pinned:     r.Pinned,
		ShowBanner: r.ShowBanner,
		Popup:      r.Popup,
		RequireAck: r.RequireAck,
		Status:     r.Status,
		StartTime:  r.StartTime,
		EndTime:    r.EndTime,
	}
}

// GetActiveNotices 用户端公告列表：只返回已发布且在生效期内的。
// 匿名访问也能拿到内容，登录用户额外带回已读/确认状态。
func GetActiveNotices(c *gin.Context) {
	notices, err := model.GetActiveNotices(c.GetInt("id"))
	if err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    notices,
	})
}

// MarkNoticeRead 用户标记公告已读；require_ack 的公告会同时记录确认时间。
func MarkNoticeRead(c *gin.Context) {
	var req struct {
		Ids []int `json:"ids"`
		Ack bool  `json:"ack"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	if err := model.MarkNoticeRead(c.GetInt("id"), req.Ids, req.Ack); err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// GetNoticesList 后台公告列表，支持关键词与状态筛选。
func GetNoticesList(c *gin.Context) {
	var params model.GenericParams
	if err := c.ShouldBindQuery(&params); err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}

	var status *int
	if raw := c.Query("status"); raw != "" && raw != "all" {
		if v, err := parseNoticeStatus(raw); err == nil {
			status = &v
		}
	}

	data, err := model.GetNoticesList(&params, status)
	if err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func parseNoticeStatus(raw string) (int, error) {
	switch raw {
	case "0":
		return model.NoticeStatusDraft, nil
	case "1":
		return model.NoticeStatusOnline, nil
	case "2":
		return model.NoticeStatusOffline, nil
	default:
		return 0, errors.New("状态无效")
	}
}

// SaveNotice 新建或更新公告（带 id 视为更新）。
func SaveNotice(c *gin.Context) {
	var req noticeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	if err := req.validate(); err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}

	notice := req.toModel()
	if notice.Id > 0 {
		if _, err := model.GetNoticeById(notice.Id); err != nil {
			common.APIRespondWithError(c, http.StatusOK, errors.New("公告不存在"))
			return
		}
		if err := model.UpdateNotice(notice); err != nil {
			common.APIRespondWithError(c, http.StatusOK, err)
			return
		}
	} else {
		notice.CreatedBy = c.GetInt("id")
		if err := model.CreateNotice(notice); err != nil {
			common.APIRespondWithError(c, http.StatusOK, err)
			return
		}
	}

	saved, err := model.GetNoticeById(notice.Id)
	if err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": saved})
}

// SetNoticeStatus 发布 / 下线 / 退回草稿。
func SetNoticeStatus(c *gin.Context) {
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	id, err := parseNoticeId(c.Param("id"))
	if err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	if err := model.SetNoticeStatus(id, req.Status); err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// DeleteNotice 删除公告（软删除，已读记录保留）。
func DeleteNotice(c *gin.Context) {
	id, err := parseNoticeId(c.Param("id"))
	if err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	if err := model.DeleteNoticeById(id); err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// GetNoticeStats 单条公告的已读与确认人数。
func GetNoticeStats(c *gin.Context) {
	id, err := parseNoticeId(c.Param("id"))
	if err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	stats, err := model.GetNoticeStats(id)
	if err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": stats})
}

func parseNoticeId(raw string) (int, error) {
	id := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, errors.New("公告 id 无效")
		}
		id = id*10 + int(ch-'0')
	}
	if id <= 0 {
		return 0, errors.New("公告 id 无效")
	}
	return id, nil
}
