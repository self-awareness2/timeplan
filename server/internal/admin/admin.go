package admin

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"timeplanner/server/internal/db"
)

type Config struct {
	DBPath string
	Token  string
}

type Service struct {
	store  *db.Store
	dbPath string
	token  string
}

type summary struct {
	UserCount       int    `json:"userCount"`
	ScheduleCount   int    `json:"scheduleCount"`
	TodayNewUsers   int    `json:"todayNewUsers"`
	TodaySchedules  int    `json:"todaySchedules"`
	DBSize          string `json:"dbSize"`
	DBPath          string `json:"dbPath"`
	Protection      string `json:"protection"`
	GeneratedAt     string `json:"generatedAt"`
	CompletedCount  int    `json:"completedCount"`
	InProgressCount int    `json:"inProgressCount"`
	PendingCount    int    `json:"pendingCount"`
}

type userRow struct {
	ID                string `json:"id"`
	Email             string `json:"email"`
	CreatedAt         string `json:"createdAt"`
	ScheduleCount     int    `json:"scheduleCount"`
	PendingCount      int    `json:"pendingCount"`
	CompletedCount    int    `json:"completedCount"`
	LastScheduleAt    string `json:"lastScheduleAt"`
	LastScheduleTitle string `json:"lastScheduleTitle"`
}

type scheduleRow struct {
	ID          int64  `json:"id"`
	Email       string `json:"email,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Date        string `json:"date"`
	Time        string `json:"time"`
	Repeat      string `json:"repeat"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	Category    string `json:"category"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type userPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewService(store *db.Store, cfg Config) *Service {
	return &Service{store: store, dbPath: cfg.DBPath, token: strings.TrimSpace(cfg.Token)}
}

func RegisterRoutes(router *gin.Engine, service *Service) {
	admin := router.Group("/admin", service.requireAdmin())
	admin.GET("", service.index)
	admin.GET("/", service.index)
	admin.GET("/api/summary", service.summary)
	admin.GET("/api/users", service.users)
	admin.POST("/api/users", service.createUser)
	admin.GET("/api/users/:id", service.userDetail)
	admin.PUT("/api/users/:id", service.updateUser)
	admin.DELETE("/api/users/:id", service.deleteUser)
	admin.GET("/api/schedules", service.schedules)
}

func (s *Service) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.token != "" {
			token := c.GetHeader("X-Admin-Token")
			if token == "" {
				token = c.Query("token")
			}
			if token != s.token {
				c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "admin token required"})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		if !isLocalRequest(c.ClientIP()) {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "error": "admin is local-only. Set TIME_PLANNER_ADMIN_TOKEN to enable remote access."})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Service) index(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = adminTemplate.Execute(c.Writer, gin.H{})
}

func (s *Service) summary(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	data := summary{
		UserCount:       s.count(`SELECT COUNT(*) FROM users`),
		ScheduleCount:   s.count(`SELECT COUNT(*) FROM schedules`),
		TodayNewUsers:   s.count(`SELECT COUNT(*) FROM users WHERE substr(created_at, 1, 10) = ?`, today),
		TodaySchedules:  s.count(`SELECT COUNT(*) FROM schedules WHERE date = ?`, today),
		DBSize:          formatFileSize(s.dbPath),
		DBPath:          s.dbPath,
		Protection:      "本机访问",
		GeneratedAt:     time.Now().Format("2006-01-02 15:04:05"),
		CompletedCount:  s.count(`SELECT COUNT(*) FROM schedules WHERE status = 'completed'`),
		InProgressCount: s.count(`SELECT COUNT(*) FROM schedules WHERE status = 'in_progress'`),
		PendingCount:    s.count(`SELECT COUNT(*) FROM schedules WHERE status = 'pending'`),
	}
	if s.token != "" {
		data.Protection = "管理员 Token"
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": data})
}

func (s *Service) users(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("q"))
	page := positiveInt(c.Query("page"), 1)
	limit := positiveInt(c.Query("limit"), 20)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	where := ""
	args := []any{}
	if keyword != "" {
		where = "WHERE users.email LIKE ?"
		args = append(args, "%"+keyword+"%")
	}

	total := s.count("SELECT COUNT(*) FROM users "+where, args...)
	queryArgs := append(args, limit, offset)
	rows, err := s.store.DB.Query(`
SELECT
  users.id,
  users.email,
  users.created_at,
  COUNT(schedules.id) AS schedule_count,
  SUM(CASE WHEN schedules.status = 'pending' THEN 1 ELSE 0 END) AS pending_count,
  SUM(CASE WHEN schedules.status = 'completed' THEN 1 ELSE 0 END) AS completed_count,
  COALESCE(MAX(schedules.updated_at), '') AS last_schedule_at
FROM users
LEFT JOIN schedules ON schedules.user_id = users.id
`+where+`
GROUP BY users.id
ORDER BY users.created_at DESC
LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	defer rows.Close()

	users := make([]userRow, 0)
	for rows.Next() {
		var user userRow
		if err := rows.Scan(&user.ID, &user.Email, &user.CreatedAt, &user.ScheduleCount, &user.PendingCount, &user.CompletedCount, &user.LastScheduleAt); err == nil {
			user.LastScheduleTitle = s.lastScheduleTitle(user.ID)
			users = append(users, user)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": gin.H{
			"items": users,
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

func (s *Service) userDetail(c *gin.Context) {
	id := c.Param("id")
	row := s.store.DB.QueryRow(`
SELECT
  users.id,
  users.email,
  users.created_at,
  COUNT(schedules.id) AS schedule_count,
  SUM(CASE WHEN schedules.status = 'pending' THEN 1 ELSE 0 END) AS pending_count,
  SUM(CASE WHEN schedules.status = 'completed' THEN 1 ELSE 0 END) AS completed_count,
  COALESCE(MAX(schedules.updated_at), '') AS last_schedule_at
FROM users
LEFT JOIN schedules ON schedules.user_id = users.id
WHERE users.id = ?
GROUP BY users.id`, id)

	var user userRow
	if err := row.Scan(&user.ID, &user.Email, &user.CreatedAt, &user.ScheduleCount, &user.PendingCount, &user.CompletedCount, &user.LastScheduleAt); err != nil {
		status := http.StatusInternalServerError
		if err == sql.ErrNoRows {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"ok": false, "error": err.Error()})
		return
	}
	user.LastScheduleTitle = s.lastScheduleTitle(user.ID)

	rows, err := s.store.DB.Query(`
SELECT id, title, description, date, start_time, end_time, repeat_type, priority, status, category, created_at, updated_at
FROM schedules
WHERE user_id = ?
ORDER BY date DESC, start_time DESC, id DESC
LIMIT 200`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	defer rows.Close()

	schedules := make([]scheduleRow, 0)
	for rows.Next() {
		var item scheduleRow
		var startTime, endTime string
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Date, &startTime, &endTime, &item.Repeat, &item.Priority, &item.Status, &item.Category, &item.CreatedAt, &item.UpdatedAt); err == nil {
			item.Time = formatTimeRange(startTime, endTime)
			schedules = append(schedules, item)
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"user": user, "schedules": schedules}})
}

func (s *Service) schedules(c *gin.Context) {
	limit := positiveInt(c.Query("limit"), 50)
	if limit > 200 {
		limit = 200
	}
	rows, err := s.store.DB.Query(`
SELECT schedules.id, users.email, schedules.title, schedules.description, schedules.date, schedules.start_time, schedules.end_time, schedules.repeat_type, schedules.priority, schedules.status, schedules.category, schedules.created_at, schedules.updated_at
FROM schedules
JOIN users ON users.id = schedules.user_id
ORDER BY schedules.updated_at DESC, schedules.id DESC
LIMIT ?`, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	defer rows.Close()

	items := make([]scheduleRow, 0)
	for rows.Next() {
		var item scheduleRow
		var startTime, endTime string
		if err := rows.Scan(&item.ID, &item.Email, &item.Title, &item.Description, &item.Date, &startTime, &endTime, &item.Repeat, &item.Priority, &item.Status, &item.Category, &item.CreatedAt, &item.UpdatedAt); err == nil {
			item.Time = formatTimeRange(startTime, endTime)
			items = append(items, item)
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"items": items, "limit": limit}})
}

func (s *Service) createUser(c *gin.Context) {
	var payload userPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请求格式不正确"})
		return
	}
	email := normalizeEmail(payload.Email)
	if email == "" || len(payload.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "邮箱不能为空，密码至少 6 位"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "密码处理失败"})
		return
	}
	id := randomID()
	_, err = s.store.DB.Exec(
		`INSERT INTO users (id, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		id, email, string(hash), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"ok": false, "error": "创建失败，邮箱可能已存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"id": id}})
}

func (s *Service) updateUser(c *gin.Context) {
	id := c.Param("id")
	var payload userPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请求格式不正确"})
		return
	}
	email := normalizeEmail(payload.Email)
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "邮箱不能为空"})
		return
	}

	if payload.Password == "" {
		result, err := s.store.DB.Exec(`UPDATE users SET email = ? WHERE id = ?`, email, id)
		respondUserUpdate(c, result, err)
		return
	}
	if len(payload.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "密码至少 6 位"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "密码处理失败"})
		return
	}
	result, err := s.store.DB.Exec(`UPDATE users SET email = ?, password_hash = ? WHERE id = ?`, email, string(hash), id)
	respondUserUpdate(c, result, err)
}

func (s *Service) deleteUser(c *gin.Context) {
	id := c.Param("id")
	tx, err := s.store.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM schedules WHERE user_id = ?`, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	result, err := tx.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "用户不存在"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"id": id}})
}

func respondUserUpdate(c *gin.Context, result sql.Result, err error) {
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"ok": false, "error": "保存失败，邮箱可能已存在"})
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"saved": true}})
}

func (s *Service) count(query string, args ...any) int {
	var count int
	if err := s.store.DB.QueryRow(query, args...).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Service) lastScheduleTitle(userID string) string {
	var title string
	_ = s.store.DB.QueryRow(`SELECT title FROM schedules WHERE user_id = ? ORDER BY updated_at DESC, id DESC LIMIT 1`, userID).Scan(&title)
	return title
}

func formatTimeRange(startTime, endTime string) string {
	if startTime == "00:00" && endTime == "00:00" {
		return "全天"
	}
	if endTime == "" || endTime == "00:00" {
		return startTime
	}
	return startTime + " - " + endTime
}

func formatFileSize(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "未知"
	}
	size := float64(info.Size())
	units := []string{"B", "KB", "MB", "GB"}
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", info.Size(), units[unit])
	}
	return fmt.Sprintf("%.1f %s", size, units[unit])
}

func positiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func randomID() string {
	bytes := make([]byte, 12)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func isLocalRequest(ipText string) bool {
	ip := net.ParseIP(ipText)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

var adminTemplate = template.Must(template.New("admin").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>TimePlanner 管理后台</title>
  <style>
    :root{--bg:#f6f7fb;--panel:#fff;--panel2:#f8fafc;--line:#e5e9f1;--text:#161a22;--muted:#667085;--accent:#3867e8;--accent-soft:#eef3ff;--good:#139a63;--warn:#c98208;--danger:#cf3d3d;--shadow:0 1px 2px rgba(16,24,40,.04),0 10px 28px rgba(16,24,40,.05)}
    *{box-sizing:border-box} body{margin:0;background:var(--bg);color:var(--text);font-family:"Segoe UI",system-ui,-apple-system,sans-serif;font-size:14px;line-height:1.45;-webkit-font-smoothing:antialiased}
    .shell{display:grid;grid-template-columns:220px minmax(0,1fr);min-height:100vh}.side{background:#0f172a;color:#cbd5e1;padding:22px 12px}.brand{color:#fff;font-size:20px;font-weight:760;margin:2px 12px 24px;letter-spacing:.01em}.nav{display:grid;gap:6px}.nav button{height:42px;border:0;border-radius:8px;background:transparent;color:inherit;text-align:left;padding:0 13px;cursor:pointer;font-weight:600}.nav button.active{background:#1e293b;color:#fff}.nav button:hover{background:#182235;color:#fff}.nav button:disabled{opacity:.38;cursor:not-allowed}.nav button:disabled:hover{background:transparent;color:inherit}
    .main{min-width:0}.top{height:68px;display:flex;align-items:center;justify-content:space-between;padding:0 28px;border-bottom:1px solid var(--line);background:rgba(255,255,255,.92);backdrop-filter:blur(10px);position:sticky;top:0;z-index:3}.top h1{font-size:20px;margin:0;font-weight:760}.muted{color:var(--muted)}.content{max-width:1480px;margin:0 auto;padding:24px 28px 36px}
    .stats{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:14px;margin-bottom:18px}.card{background:var(--panel);border:1px solid var(--line);border-radius:8px;box-shadow:var(--shadow)}.stat{padding:17px 18px}.stat .label{color:var(--muted);font-size:13px}.stat .num{font-size:30px;font-weight:760;margin-top:6px;line-height:1}.grid{display:grid;grid-template-columns:1fr;gap:16px;align-items:start}
    .toolbar{display:flex;gap:10px;padding:14px 16px;border-bottom:1px solid var(--line);align-items:center}.toolbar input{height:40px;flex:1;min-width:220px;border:1px solid var(--line);border-radius:8px;padding:0 12px;outline:none;background:#fff}.toolbar input:focus,.field input:focus{border-color:var(--accent);box-shadow:0 0 0 3px rgba(56,103,232,.12)}.btn{height:38px;border:1px solid var(--line);border-radius:8px;background:#fff;padding:0 13px;cursor:pointer;white-space:nowrap;font-weight:600}.btn.primary{background:var(--accent);border-color:var(--accent);color:#fff}.btn.danger{border-color:#f2c6c6;color:var(--danger);background:#fff7f7}.btn.small{height:30px;padding:0 9px;font-size:12px}.btn:disabled{opacity:.45;cursor:not-allowed}
    .table-wrap{overflow:auto}table{width:100%;min-width:980px;border-collapse:collapse;table-layout:fixed}th,td{padding:12px 14px;border-bottom:1px solid var(--line);text-align:left;vertical-align:middle}th{font-size:12px;color:var(--muted);font-weight:700;background:var(--panel2);white-space:nowrap}td{white-space:nowrap;overflow:hidden;text-overflow:ellipsis}th:nth-child(1){width:310px}th:nth-child(2),th:nth-child(3),th:nth-child(4){width:78px}th:nth-child(5){width:180px}th:nth-child(6){width:170px}th:nth-child(7){width:130px}tr.user-row{cursor:pointer}tr.user-row:hover{background:#f8fbff}tr.user-row.active{background:var(--accent-soft)}.email{font-weight:680}.pill{display:inline-flex;align-items:center;justify-content:center;min-width:26px;min-height:24px;border-radius:999px;padding:2px 8px;background:#f1f5f9;color:#475569;font-size:12px}.pill.good{background:#e9f8f1;color:var(--good)}.pill.warn{background:#fff6df;color:var(--warn)}
    .panel-head{display:flex;align-items:center;justify-content:space-between;padding:14px 16px;border-bottom:1px solid var(--line)}.panel-head h2{margin:0;font-size:15px}.detail{padding:16px}.kv{display:grid;grid-template-columns:92px minmax(0,1fr);gap:9px 12px;margin-bottom:14px}.schedule-list{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:10px;max-height:420px;overflow:auto}.schedule{border:1px solid var(--line);border-radius:8px;padding:11px;background:#fff}.schedule-title{font-weight:680;margin-bottom:5px}.schedule-meta{display:flex;gap:6px;flex-wrap:wrap;color:var(--muted);font-size:12px}.empty{padding:42px 24px;color:var(--muted);text-align:center}.footer{display:flex;align-items:center;justify-content:space-between;padding:12px 14px}
    .dbline{margin-top:16px;padding:12px 14px;background:#fff;border:1px solid var(--line);border-radius:8px;color:var(--muted);font-size:12px;overflow:auto}.actions{display:flex;gap:6px}.view{display:none}.view.active{display:block}.section-title{display:flex;align-items:center;justify-content:space-between;margin:0 0 14px}.section-title h2{margin:0;font-size:18px}.mini-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:14px}.info-card{padding:16px}.info-card strong{display:block;margin-bottom:6px}.modal-mask{position:fixed;inset:0;display:none;align-items:center;justify-content:center;background:rgba(15,23,42,.42);z-index:20;padding:16px}.modal-mask.show{display:flex}.modal{width:min(420px,100%);background:#fff;border-radius:12px;border:1px solid var(--line);box-shadow:0 24px 70px rgba(15,23,42,.25)}.modal-head,.modal-foot{display:flex;align-items:center;justify-content:space-between;padding:16px 18px;border-bottom:1px solid var(--line)}.modal-foot{border-top:1px solid var(--line);border-bottom:0;justify-content:flex-end;gap:8px}.modal-body{padding:18px;display:grid;gap:12px}.field{display:grid;gap:6px}.field label{color:var(--muted);font-size:13px}.field input{height:40px;border:1px solid var(--line);border-radius:8px;padding:0 11px;outline:none}.hint{font-size:12px;color:var(--muted)}
    @media(max-width:980px){.shell{grid-template-columns:1fr}.side{display:none}.top{padding:0 16px}.content{padding:16px}.stats{grid-template-columns:1fr 1fr}.toolbar{flex-wrap:wrap}.toolbar input{flex-basis:100%}.schedule-list{grid-template-columns:1fr}}
  </style>
</head>
<body>
  <div class="shell">
    <aside class="side">
      <div class="brand">TimePlanner</div>
      <div class="nav">
        <button class="active" data-view="users">用户管理</button>
        <button data-view="plans">计划数据</button>
        <button data-view="overview">运营概览</button>
        <button data-view="settings">系统设置</button>
      </div>
    </aside>
    <main class="main">
      <header class="top">
        <div><h1>用户管理</h1><div class="muted" id="generated">正在加载数据...</div></div>
        <button class="btn primary" id="refreshBtn">刷新</button>
      </header>
      <section class="content">
        <div class="view active" id="view-users">
        <div class="stats">
          <div class="card stat"><div class="label">总用户</div><div class="num" id="userCount">-</div></div>
          <div class="card stat"><div class="label">总计划</div><div class="num" id="scheduleCount">-</div></div>
          <div class="card stat"><div class="label">今日新增用户</div><div class="num" id="todayUsers">-</div></div>
          <div class="card stat"><div class="label">今日计划</div><div class="num" id="todaySchedules">-</div></div>
        </div>
        <div class="grid">
          <section class="card">
            <div class="toolbar">
              <input id="search" placeholder="搜索用户邮箱..." />
              <button class="btn" id="searchBtn">搜索</button>
              <button class="btn primary" id="createBtn">新增用户</button>
            </div>
            <div class="table-wrap">
              <table>
                <thead><tr><th>用户</th><th>计划</th><th>完成</th><th>待办</th><th>最近活动</th><th>注册时间</th><th>操作</th></tr></thead>
                <tbody id="usersBody"><tr><td colspan="7" class="empty">加载中...</td></tr></tbody>
              </table>
            </div>
            <div class="footer">
              <span class="muted" id="pageInfo">-</span>
              <div><button class="btn" id="prevBtn">上一页</button> <button class="btn" id="nextBtn">下一页</button></div>
            </div>
          </section>
          <aside class="card detail-card">
            <div class="panel-head"><h2>用户详情</h2><span class="pill" id="detailState">未选择</span></div>
            <div class="detail" id="detail"><div class="empty">点击左侧用户查看账号和计划明细</div></div>
          </aside>
        </div>
        <div class="dbline" id="dbline"></div>
        </div>
        <div class="view" id="view-plans">
          <div class="section-title"><h2>计划数据</h2><button class="btn" id="reloadPlansBtn">刷新计划</button></div>
          <section class="card">
            <div class="table-wrap">
              <table>
                <thead><tr><th>ID</th><th>用户</th><th>标题</th><th>日期</th><th>时间</th><th>状态</th><th>优先级</th><th>更新</th></tr></thead>
                <tbody id="plansBody"><tr><td colspan="8" class="empty">点击刷新计划加载数据</td></tr></tbody>
              </table>
            </div>
          </section>
        </div>
        <div class="view" id="view-overview">
          <div class="section-title"><h2>运营概览</h2><button class="btn" id="reloadOverviewBtn">刷新概览</button></div>
          <div class="mini-grid">
            <div class="card info-card"><strong>用户规模</strong><div class="muted" id="overviewUsers">-</div></div>
            <div class="card info-card"><strong>计划活跃</strong><div class="muted" id="overviewPlans">-</div></div>
            <div class="card info-card"><strong>完成情况</strong><div class="muted" id="overviewDone">-</div></div>
            <div class="card info-card"><strong>数据文件</strong><div class="muted" id="overviewDb">-</div></div>
          </div>
        </div>
        <div class="view" id="view-settings">
          <div class="section-title"><h2>系统设置</h2></div>
          <div class="mini-grid">
            <div class="card info-card"><strong>管理员访问</strong><div class="muted">默认仅本机/内网可访问；公网使用时建议设置 TIME_PLANNER_ADMIN_TOKEN。</div></div>
            <div class="card info-card"><strong>数据库</strong><div class="muted" id="settingsDb">-</div></div>
            <div class="card info-card"><strong>后台权限</strong><div class="muted">当前版本支持用户增删改查和只读计划查看。</div></div>
          </div>
        </div>
      </section>
    </main>
  </div>
  <div class="modal-mask" id="userModal">
    <div class="modal">
      <div class="modal-head"><strong id="modalTitle">新增用户</strong><button class="btn small" id="modalClose">关闭</button></div>
      <div class="modal-body">
        <input type="hidden" id="userId" />
        <div class="field"><label>邮箱 / 账号</label><input id="userEmail" autocomplete="off" /></div>
        <div class="field"><label>密码</label><input id="userPassword" type="password" autocomplete="new-password" /></div>
        <div class="hint" id="passwordHint">新增用户时密码至少 6 位。</div>
      </div>
      <div class="modal-foot">
        <button class="btn" id="modalCancel">取消</button>
        <button class="btn primary" id="saveUserBtn">保存</button>
      </div>
    </div>
  </div>
  <script>
    const token = new URLSearchParams(location.search).get('token') || '';
    const withToken = (path) => token ? path + (path.includes('?') ? '&' : '?') + 'token=' + encodeURIComponent(token) : path;
    let page = 1, limit = 20, total = 0, query = '', selectedId = '', lastSummary = null;

    async function api(path, options = {}) {
      const headers = Object.assign({ 'Content-Type': 'application/json' }, token ? { 'X-Admin-Token': token } : {}, options.headers || {});
      const res = await fetch(withToken(path), Object.assign({}, options, { headers }));
      const json = await res.json();
      if (!json.ok) throw new Error(json.error || '请求失败');
      return json.data;
    }
    const text = (id, value) => document.getElementById(id).textContent = value;
    const esc = (s) => String(s ?? '').replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
    const fmt = (s) => s ? s.replace('T', ' ').slice(0, 19) : '-';
    const statusLabel = { pending:'待办', in_progress:'进行中', completed:'已完成', cancelled:'已取消' };
    const priorityLabel = { low:'低', medium:'中', high:'高' };

    async function loadSummary() {
      const s = await api('/admin/api/summary');
      lastSummary = s;
      text('userCount', s.userCount);
      text('scheduleCount', s.scheduleCount);
      text('todayUsers', s.todayNewUsers);
      text('todaySchedules', s.todaySchedules);
      text('generated', '更新时间 ' + s.generatedAt + ' · ' + s.protection);
      text('dbline', '数据库：' + s.dbPath + ' · 大小：' + s.dbSize + ' · 待办 ' + s.pendingCount + ' · 进行中 ' + s.inProgressCount + ' · 已完成 ' + s.completedCount);
      updateOverview();
      text('settingsDb', s.dbPath + ' · ' + s.dbSize);
    }

    async function loadUsers() {
      const data = await api('/admin/api/users?page=' + page + '&limit=' + limit + '&q=' + encodeURIComponent(query));
      total = data.total;
      const body = document.getElementById('usersBody');
      if (!data.items.length) {
        body.innerHTML = '<tr><td colspan="7" class="empty">没有找到用户</td></tr>';
      } else {
        body.innerHTML = data.items.map(u => '<tr class="user-row ' + (u.id === selectedId ? 'active' : '') + '" data-id="' + esc(u.id) + '">' +
          '<td><div class="email">' + esc(u.email) + '</div><div class="muted">' + esc(u.id) + '</div></td>' +
          '<td><span class="pill">' + u.scheduleCount + '</span></td>' +
          '<td><span class="pill good">' + u.completedCount + '</span></td>' +
          '<td><span class="pill warn">' + u.pendingCount + '</span></td>' +
          '<td><div>' + esc(u.lastScheduleTitle || '-') + '</div><div class="muted">' + fmt(u.lastScheduleAt) + '</div></td>' +
          '<td>' + fmt(u.createdAt) + '</td>' +
          '<td><div class="actions"><button class="btn small" data-edit="' + esc(u.id) + '" data-email="' + esc(u.email) + '">编辑</button><button class="btn small danger" data-delete="' + esc(u.id) + '" data-email="' + esc(u.email) + '">删除</button></div></td></tr>').join('');
      }
      text('pageInfo', '第 ' + page + ' 页，共 ' + total + ' 个用户');
      document.getElementById('prevBtn').disabled = page <= 1;
      document.getElementById('nextBtn').disabled = page * limit >= total;
      body.querySelectorAll('tr[data-id]').forEach(row => row.addEventListener('click', (event) => {
        if (event.target.closest('button')) return;
        loadDetail(row.dataset.id);
      }));
      body.querySelectorAll('[data-edit]').forEach(btn => btn.addEventListener('click', () => openUserModal(btn.dataset.edit, btn.dataset.email)));
      body.querySelectorAll('[data-delete]').forEach(btn => btn.addEventListener('click', () => deleteUser(btn.dataset.delete, btn.dataset.email)));
    }

    async function loadDetail(id) {
      selectedId = id;
      await loadUsers();
      const data = await api('/admin/api/users/' + encodeURIComponent(id));
      text('detailState', '已选择');
      const u = data.user;
      document.getElementById('detail').innerHTML =
        '<div class="kv"><div class="muted">邮箱</div><div class="email">' + esc(u.email) + '</div>' +
        '<div class="muted">用户ID</div><div>' + esc(u.id) + '</div>' +
        '<div class="muted">注册时间</div><div>' + fmt(u.createdAt) + '</div>' +
        '<div class="muted">计划数量</div><div>' + u.scheduleCount + ' 项</div></div>' +
        '<div class="schedule-list">' + (data.schedules.length ? data.schedules.map(renderSchedule).join('') : '<div class="empty">该用户暂无计划</div>') + '</div>';
    }

    function renderSchedule(s) {
      return '<div class="schedule"><div class="schedule-title">' + esc(s.title) + '</div>' +
        '<div class="schedule-meta"><span>' + esc(s.date) + '</span><span>' + esc(s.time) + '</span><span>' + esc(statusLabel[s.status] || s.status) + '</span><span>' + esc(priorityLabel[s.priority] || s.priority) + '</span>' + (s.category ? '<span>' + esc(s.category) + '</span>' : '') + '</div>' +
        (s.description ? '<div class="muted" style="margin-top:6px">' + esc(s.description) + '</div>' : '') + '</div>';
    }

    async function loadPlans() {
      const data = await api('/admin/api/schedules?limit=100');
      const body = document.getElementById('plansBody');
      if (!data.items.length) {
        body.innerHTML = '<tr><td colspan="8" class="empty">暂无计划数据</td></tr>';
        return;
      }
      body.innerHTML = data.items.map(s => '<tr>' +
        '<td>' + s.id + '</td>' +
        '<td title="' + esc(s.email || '-') + '">' + esc(s.email || '-') + '</td>' +
        '<td title="' + esc(s.title) + '">' + esc(s.title) + '</td>' +
        '<td>' + esc(s.date) + '</td>' +
        '<td>' + esc(s.time) + '</td>' +
        '<td><span class="pill">' + esc(statusLabel[s.status] || s.status) + '</span></td>' +
        '<td><span class="pill">' + esc(priorityLabel[s.priority] || s.priority) + '</span></td>' +
        '<td>' + fmt(s.updatedAt) + '</td></tr>').join('');
    }

    function updateOverview() {
      if (!lastSummary) return;
      text('overviewUsers', '总用户 ' + lastSummary.userCount + '，今日新增 ' + lastSummary.todayNewUsers);
      text('overviewPlans', '总计划 ' + lastSummary.scheduleCount + '，今日计划 ' + lastSummary.todaySchedules);
      text('overviewDone', '待办 ' + lastSummary.pendingCount + '，进行中 ' + lastSummary.inProgressCount + '，已完成 ' + lastSummary.completedCount);
      text('overviewDb', lastSummary.dbPath + ' · ' + lastSummary.dbSize);
    }

    function switchView(name) {
      document.querySelectorAll('.nav button[data-view]').forEach(btn => btn.classList.toggle('active', btn.dataset.view === name));
      document.querySelectorAll('.view').forEach(view => view.classList.remove('active'));
      document.getElementById('view-' + name).classList.add('active');
      const titles = { users:'用户管理', plans:'计划数据', overview:'运营概览', settings:'系统设置' };
      document.querySelector('.top h1').textContent = titles[name] || '管理后台';
      if (name === 'plans') loadPlans().catch(err => alert(err.message));
      if (name === 'overview') updateOverview();
    }

    function openUserModal(id = '', email = '') {
      document.getElementById('userId').value = id;
      document.getElementById('userEmail').value = email;
      document.getElementById('userPassword').value = '';
      text('modalTitle', id ? '编辑用户' : '新增用户');
      text('passwordHint', id ? '不填写密码则保持原密码不变。' : '新增用户时密码至少 6 位。');
      document.getElementById('userModal').classList.add('show');
      document.getElementById('userEmail').focus();
    }

    function closeUserModal() {
      document.getElementById('userModal').classList.remove('show');
    }

    async function saveUser() {
      const id = document.getElementById('userId').value;
      const email = document.getElementById('userEmail').value.trim();
      const password = document.getElementById('userPassword').value;
      if (!email) return alert('请填写邮箱 / 账号');
      if (!id && password.length < 6) return alert('新增用户密码至少 6 位');
      const body = JSON.stringify({ email, password });
      if (id) {
        await api('/admin/api/users/' + encodeURIComponent(id), { method: 'PUT', body });
      } else {
        await api('/admin/api/users', { method: 'POST', body });
        page = 1;
      }
      closeUserModal();
      await refresh();
      if (id) await loadDetail(id);
    }

    async function deleteUser(id, email) {
      if (!confirm('确定删除用户 ' + email + '？该用户的计划也会一起删除。')) return;
      await api('/admin/api/users/' + encodeURIComponent(id), { method: 'DELETE' });
      if (selectedId === id) {
        selectedId = '';
        text('detailState', '未选择');
        document.getElementById('detail').innerHTML = '<div class="empty">点击左侧用户查看账号和计划明细</div>';
      }
      await refresh();
    }

    async function refresh() { await Promise.all([loadSummary(), loadUsers()]); }
    document.getElementById('refreshBtn').onclick = refresh;
    document.querySelectorAll('.nav button[data-view]').forEach(btn => btn.addEventListener('click', () => switchView(btn.dataset.view)));
    document.getElementById('reloadPlansBtn').onclick = () => loadPlans().catch(err => alert(err.message));
    document.getElementById('reloadOverviewBtn').onclick = () => loadSummary().catch(err => alert(err.message));
    document.getElementById('createBtn').onclick = () => openUserModal();
    document.getElementById('modalClose').onclick = closeUserModal;
    document.getElementById('modalCancel').onclick = closeUserModal;
    document.getElementById('saveUserBtn').onclick = () => saveUser().catch(err => alert(err.message));
    document.getElementById('userModal').addEventListener('click', e => { if (e.target.id === 'userModal') closeUserModal(); });
    document.getElementById('searchBtn').onclick = () => { query = document.getElementById('search').value.trim(); page = 1; selectedId = ''; loadUsers(); };
    document.getElementById('search').addEventListener('keydown', e => { if (e.key === 'Enter') document.getElementById('searchBtn').click(); });
    document.getElementById('prevBtn').onclick = () => { if (page > 1) { page--; loadUsers(); } };
    document.getElementById('nextBtn').onclick = () => { if (page * limit < total) { page++; loadUsers(); } };
    refresh().catch(err => { document.body.innerHTML = '<pre style="padding:24px;color:#b42318">' + esc(err.message) + '</pre>'; });
  </script>
</body>
</html>`))
