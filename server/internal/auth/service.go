package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"chrona/server/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	store  *db.Store
	secret []byte
}

type User struct {
	ID       string `json:"-"`
	Username string `json:"username"`
}

func NewService(store *db.Store, secret string) *Service {
	return &Service{store: store, secret: []byte(secret)}
}

func RegisterRoutes(group *gin.RouterGroup, service *Service) {
	group.POST("/register", service.register)
	group.POST("/login", service.login)
	group.GET("/me", service.RequireUser(), func(c *gin.Context) {
		user := CurrentUser(c)
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"user": gin.H{"username": user.Username}}})
	})
}

func (s *Service) register(c *gin.Context) {
	var req credentials
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "code": "invalid_request", "error": "invalid request"})
		return
	}
	username := normalizeUsername(req.Username)
	if username == "" || len(username) > 32 || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "code": "invalid_request", "error": "invalid request"})
		return
	}
	if _, err := s.findUserByUsername(username); err == nil {
		c.JSON(http.StatusConflict, gin.H{"ok": false, "code": "username_taken", "error": "username already exists"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "code": "server_error", "error": "server error"})
		return
	}
	user := User{ID: randomID(), Username: username}
	_, err = s.store.DB.Exec(
		`INSERT INTO users (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		user.ID, user.Username, string(hash), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "code": "server_error", "error": "server error"})
		return
	}
	s.respondSession(c, user)
}

func (s *Service) login(c *gin.Context) {
	var req credentials
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "code": "invalid_request", "error": "invalid request"})
		return
	}
	row := s.store.DB.QueryRow(`SELECT id, username, password_hash FROM users WHERE username = ?`, normalizeUsername(req.Username))
	var user User
	var hash string
	if err := row.Scan(&user.ID, &user.Username, &hash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "code": "invalid_credentials", "error": "username or password is incorrect"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "code": "invalid_credentials", "error": "username or password is incorrect"})
		return
	}
	s.respondSession(c, user)
}

func (s *Service) RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenText := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenText == "" {
			if cookie, err := c.Request.Cookie("chrona_session"); err == nil {
				tokenText = cookie.Value
			}
		}
		userID, err := s.verifyToken(tokenText)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "code": "auth_required", "error": "authentication required"})
			return
		}
		user, err := s.findUserByID(userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "code": "session_expired", "error": "session expired"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "code": "server_error", "error": "server error"})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: "chrona_session", Value: token, Path: "/", MaxAge: 24 * 60 * 60,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: c.Request.TLS != nil,
	})
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"token": token, "user": gin.H{"username": user.Username}}})
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

func (s *Service) findUserByUsername(username string) (User, error) {
	row := s.store.DB.QueryRow(`SELECT id, username FROM users WHERE username = ?`, username)
	var user User
	err := row.Scan(&user.ID, &user.Username)
	return user, err
}

func (s *Service) findUserByID(id string) (User, error) {
	row := s.store.DB.QueryRow(`SELECT id, username FROM users WHERE id = ?`, id)
	var user User
	err := row.Scan(&user.ID, &user.Username)
	if err == sql.ErrNoRows {
		return User{}, err
	}
	return user, err
}

func normalizeUsername(username string) string {
	return strings.TrimSpace(username)
}

func randomID() string {
	bytes := make([]byte, 12)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
