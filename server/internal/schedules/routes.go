package schedules

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"timeplanner/server/internal/auth"
)

func RegisterRoutes(group *gin.RouterGroup, authService *auth.Service, service *Service) {
	group.POST("", authService.RequireUser(), func(c *gin.Context) {
		var req ActionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请求格式不正确"})
			return
		}
		user := auth.CurrentUser(c)
		data, err := service.Dispatch(user.ID, req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": data})
	})
}
