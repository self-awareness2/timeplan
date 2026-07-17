package schedules

import (
	"net/http"

	"chrona/server/internal/auth"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(group *gin.RouterGroup, authService *auth.Service, service *Service) {
	group.GET("/export", authService.RequireUser(), func(c *gin.Context) {
		user := auth.CurrentUser(c)
		data, err := service.Export(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "code": "export_failed", "error": "export failed"})
			return
		}
		c.Header("Content-Disposition", `attachment; filename="chrona-export.json"`)
		c.JSON(http.StatusOK, data)
	})
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
