package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 公告状态：草稿只在后台可见；下线保留内容但不再对用户展示。
const (
	NoticeStatusDraft   = 0
	NoticeStatusOnline  = 1
	NoticeStatusOffline = 2
)

// 公告级别决定前端配色与图标。
const (
	NoticeLevelInfo    = "info"
	NoticeLevelSuccess = "success"
	NoticeLevelWarning = "warning"
	NoticeLevelError   = "error"
)

// Notice 一条公告。取代原先 options.Notice 只能存一段文本的方案，
// 支持多条并存、定时生效、置顶、横幅/弹窗开关与强制确认。
type Notice struct {
	Id         int            `json:"id"`
	Title      string         `json:"title" gorm:"type:varchar(200);not null"`
	Content    string         `json:"content" gorm:"type:text"`
	Level      string         `json:"level" gorm:"type:varchar(16);default:'info'"`
	Pinned     bool           `json:"pinned" gorm:"default:false"`
	ShowBanner bool           `json:"show_banner" gorm:"default:false"`
	Popup      bool           `json:"popup" gorm:"default:true"`
	RequireAck bool           `json:"require_ack" gorm:"default:false"`
	Status     int            `json:"status" gorm:"default:0;index"`
	StartTime  int64          `json:"start_time" gorm:"bigint;default:0"`
	EndTime    int64          `json:"end_time" gorm:"bigint;default:0"`
	ReadCount  int            `json:"read_count" gorm:"default:0"`
	AckCount   int            `json:"ack_count" gorm:"default:0"`
	CreatedBy  int            `json:"created_by" gorm:"default:0"`
	CreatedAt  int64          `json:"created_at" gorm:"bigint"`
	UpdatedAt  int64          `json:"updated_at" gorm:"bigint"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`

	// 以下字段不落库，按当前访问者实时计算。
	Read  bool `json:"read" gorm:"-"`
	Acked bool `json:"acked" gorm:"-"`
}

// NoticeRead 记录某用户对某条公告的已读/确认状态。
// 同一条公告每个用户只保留一行，重复标记时刷新时间。
type NoticeRead struct {
	Id       int   `json:"id"`
	UserId   int   `json:"user_id" gorm:"uniqueIndex:idx_notice_read_user"`
	NoticeId int   `json:"notice_id" gorm:"uniqueIndex:idx_notice_read_user"`
	ReadAt   int64 `json:"read_at" gorm:"bigint;default:0"`
	AckAt    int64 `json:"ack_at" gorm:"bigint;default:0"`
}

func (NoticeRead) TableName() string {
	return "notice_reads"
}

var allowedNoticeOrderFields = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
	"status":     true,
	"pinned":     true,
}

// GetNoticesList 管理端列表：默认全部状态，可按关键词与状态筛选。
func GetNoticesList(params *GenericParams, status *int) (*DataResult[Notice], error) {
	var notices []*Notice
	db := DB.Model(&Notice{})
	if params.Keyword != "" {
		db = db.Where("title LIKE ? OR content LIKE ?", "%"+params.Keyword+"%", "%"+params.Keyword+"%")
	}
	if status != nil {
		db = db.Where("status = ?", *status)
	}
	return PaginateAndOrder[Notice](db, &params.PaginationParams, &notices, allowedNoticeOrderFields)
}

func GetNoticeById(id int) (*Notice, error) {
	if id <= 0 {
		return nil, errors.New("公告 id 无效")
	}
	notice := Notice{}
	if err := DB.First(&notice, id).Error; err != nil {
		return nil, err
	}
	return &notice, nil
}

func CreateNotice(notice *Notice) error {
	notice.Id = 0
	return DB.Create(notice).Error
}

func UpdateNotice(notice *Notice) error {
	if notice.Id <= 0 {
		return errors.New("公告 id 无效")
	}
	// 显式指定字段：避免零值（false / 0）被 GORM 当成「未提供」而跳过。
	return DB.Model(&Notice{Id: notice.Id}).Select(
		"title", "content", "level", "pinned", "show_banner", "popup",
		"require_ack", "status", "start_time", "end_time",
	).Updates(notice).Error
}

func DeleteNoticeById(id int) error {
	return DB.Delete(&Notice{}, id).Error
}

// SetNoticeStatus 发布（1）、下线（2）或退回草稿（0）。
func SetNoticeStatus(id int, status int) error {
	if status != NoticeStatusOnline && status != NoticeStatusOffline && status != NoticeStatusDraft {
		return errors.New("状态值无效")
	}
	res := DB.Model(&Notice{}).Where("id = ?", id).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("公告不存在")
	}
	return nil
}

// GetActiveNotices 返回当前对所有访问者可见的公告：
// 已发布、且在生效时间窗内（0 表示不限）。
func GetActiveNotices(userId int) ([]*Notice, error) {
	now := time.Now().Unix()
	notices := make([]*Notice, 0)
	err := DB.Where("status = ?", NoticeStatusOnline).
		Where("(start_time = 0 OR start_time <= ?)", now).
		Where("(end_time = 0 OR end_time >= ?)", now).
		Order("pinned DESC, id DESC").
		Find(&notices).Error
	if err != nil {
		return nil, err
	}
	if userId <= 0 || len(notices) == 0 {
		return notices, nil
	}

	ids := make([]int, 0, len(notices))
	for _, n := range notices {
		ids = append(ids, n.Id)
	}
	var reads []NoticeRead
	if err := DB.Where("user_id = ? AND notice_id IN ?", userId, ids).Find(&reads).Error; err != nil {
		return notices, nil
	}
	state := make(map[int]NoticeRead, len(reads))
	for _, r := range reads {
		state[r.NoticeId] = r
	}
	for _, n := range notices {
		if r, ok := state[n.Id]; ok {
			n.Read = r.ReadAt > 0
			n.Acked = r.AckAt > 0
		}
	}
	return notices, nil
}

// MarkNoticeRead 标记已读；ack=true 时同时记录确认时间。
// 只处理真实存在的公告，避免伪造 id 写脏数据。
func MarkNoticeRead(userId int, ids []int, ack bool) error {
	if userId <= 0 {
		return errors.New("用户未登录")
	}
	if len(ids) == 0 {
		return nil
	}

	valid := make([]int, 0, len(ids))
	if err := DB.Model(&Notice{}).Where("id IN ?", ids).Pluck("id", &valid).Error; err != nil {
		return err
	}
	if len(valid) == 0 {
		return nil
	}

	now := time.Now().Unix()
	for _, id := range valid {
		var existing NoticeRead
		err := DB.Where("user_id = ? AND notice_id = ?", userId, id).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			row := NoticeRead{UserId: userId, NoticeId: id, ReadAt: now}
			if ack {
				row.AckAt = now
			}
			if e := DB.Create(&row).Error; e != nil {
				// 并发下可能已被别的请求插入，唯一键冲突可直接忽略。
				continue
			}
		case err == nil:
			fields := map[string]any{"read_at": now}
			if ack {
				fields["ack_at"] = now
			}
			DB.Model(&NoticeRead{}).Where("id = ?", existing.Id).Updates(fields)
		}
	}

	return refreshNoticeCounters(valid)
}

// refreshNoticeCounters 重算已读/确认人数，冗余到 notices 表供列表直接展示。
func refreshNoticeCounters(ids []int) error {
	for _, id := range ids {
		var readCount, ackCount int64
		DB.Model(&NoticeRead{}).Where("notice_id = ? AND read_at > 0", id).Count(&readCount)
		DB.Model(&NoticeRead{}).Where("notice_id = ? AND ack_at > 0", id).Count(&ackCount)
		if err := DB.Model(&Notice{}).Where("id = ?", id).
			Updates(map[string]any{"read_count": readCount, "ack_count": ackCount}).Error; err != nil {
			return err
		}
	}
	return nil
}

// NoticeStats 单条公告的阅读情况，用于后台详情。
type NoticeStats struct {
	NoticeId    int   `json:"notice_id"`
	ReadCount   int64 `json:"read_count"`
	AckCount    int64 `json:"ack_count"`
	UserCount   int64 `json:"user_count"`
	UnreadCount int64 `json:"unread_count"`
	UpdatedAt   int64 `json:"updated_at"`
}

func GetNoticeStats(id int) (*NoticeStats, error) {
	notice, err := GetNoticeById(id)
	if err != nil {
		return nil, err
	}
	stats := &NoticeStats{NoticeId: id, UpdatedAt: notice.UpdatedAt}
	DB.Model(&NoticeRead{}).Where("notice_id = ? AND read_at > 0", id).Count(&stats.ReadCount)
	DB.Model(&NoticeRead{}).Where("notice_id = ? AND ack_at > 0", id).Count(&stats.AckCount)
	DB.Model(&User{}).Where("status = ?", 1).Count(&stats.UserCount)
	if stats.UserCount > stats.ReadCount {
		stats.UnreadCount = stats.UserCount - stats.ReadCount
	}
	return stats, nil
}

// MigrateLegacyNotice 把旧版 options.Notice 里的一段 Markdown 转成第一条公告。
// 仅在公告表为空且旧内容非空时执行，重复调用没有副作用。
func MigrateLegacyNotice(tx *gorm.DB) error {
	var count int64
	if err := tx.Model(&Notice{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var option Option
	if err := tx.Where(&Option{Key: "Notice"}).First(&option).Error; err != nil {
		return nil
	}
	content := strings.TrimSpace(option.Value)
	if content == "" {
		return nil
	}

	title := "站点公告"
	body := content
	// 旧公告若以 # 标题开头，取第一行作为标题，正文去掉该行。
	lines := strings.SplitN(content, "\n", 2)
	if first := strings.TrimSpace(lines[0]); strings.HasPrefix(first, "#") {
		if t := strings.TrimSpace(strings.TrimLeft(first, "#")); t != "" {
			title = t
			if len(lines) > 1 {
				body = strings.TrimSpace(lines[1])
			} else {
				body = ""
			}
		}
	}

	notice := Notice{
		Title:   title,
		Content: body,
		Level:   NoticeLevelInfo,
		Pinned:  true,
		// 迁移过来的旧公告用户基本已经看过，只放进收件箱，不再弹窗打扰。
		Popup:     false,
		Status:    NoticeStatusOnline,
		CreatedBy: 1,
	}
	return tx.Create(&notice).Error
}
