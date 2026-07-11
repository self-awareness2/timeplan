package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"timeplanner/server/internal/db"
)

type Service struct {
	store  *db.Store
	secret []byte
}

type User struct {
	ID    string `json:"-"`
	Email string `json:"email"`
}

func NewService(store *db.Store, secret string) *Service {
	return &Service{store: store, secret: []byte(secret)}
}

func RegisterRoutes(group *gin.RouterGroup, service *Service) {
	group.POST("/register", service.register)
	group.POST("/login", service.login)
	group.GET("/me", service.RequireUser(), func(c *gin.Context) {
		user := CurrentUser(c)
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"user": gin.H{"email": user.Email}}})
	})
}

func (s *Service) register(c *gin.Context) {
	var req credentials
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请求格式不正确"})
		return
	}
	email := normalizeEmail(req.Email)
	if email == "" || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "邮箱不能为空，密码至少 6 位"})
		return
	}
	if _, err := s.findUserByEmail(email); err == nil {
		c.JSON(http.StatusConflict, gin.H{"ok": false, "error": "账号已存在"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "密码处理失败"})
		return
	}
	user := User{ID: randomID(), Email: email}
	_, err = s.store.DB.Exec(
		`INSERT INTO users (id, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		user.ID, user.Email, string(hash), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "创建账号失败"})
		return
	}
	s.respondSession(c, user)
}

func (s *Service) login(c *gin.Context) {
	var req credentials
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请求格式不正确"})
		return
	}
	row := s.store.DB.QueryRow(`SELECT id, email, password_hash FROM users WHERE email = ?`, normalizeEmail(req.Email))
	var user User
	var hash string
	if err := row.Scan(&user.ID, &user.Email, &hash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "邮箱或密码不正确"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "邮箱或密码不正确"})
		return
	}
	s.respondSession(c, user)
}

func (s *Service) RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenText := strings.TrimPrefix(authHeader, "Bearer ")
		userID, err := s.verifyToken(tokenText)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "请先登录"})
			return
		}
		user, err := s.findUserByID(userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "登录已失效"})
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) User {
	user, _ := c.Get("user")
	return user.(User)
}

func (s *Service) respondSession(c *gin.Context, user User) {
	token, err := s.makeToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "生成登录态失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"token": token, "user": gin.H{"email": user.Email}}})
}

func (s *Service) makeToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"userId": userID,
		"exp":    time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *Service) verifyToken(tokenText string) (string, error) {
	token, err := jwt.Parse(tokenText, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	userID, ok := claims["userId"].(string)
	if !ok || userID == "" {
		return "", errors.New("missing user id")
	}
	return userID, nil
}

func (s *Service) findUserByEmail(email string) (User, error) {
	row := s.store.DB.QueryRow(`SELECT id, email FROM users WHERE email = ?`, email)
	var user User
	err := row.Scan(&user.ID, &user.Email)
	return user, err
}

func (s *Service) findUserByID(id string) (User, error) {
	row := s.store.DB.QueryRow(`SELECT id, email FROM users WHERE id = ?`, id)
	var user User
	err := row.Scan(&user.ID, &user.Email)
	if err == sql.ErrNoRows {
		return User{}, err
	}
	return user, err
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func randomID() string {
	bytes := make([]byte, 12)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
