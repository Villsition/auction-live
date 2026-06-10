package handler

import (
	"net/http"

	"auction/pkg/errcode"
	redisPkg "auction/pkg/redis"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db  *gorm.DB
	rdb *redisPkg.Client
}

func NewHealthHandler(db *gorm.DB, rdb *redisPkg.Client) *HealthHandler {
	return &HealthHandler{db: db, rdb: rdb}
}

// Check returns the health status of MySQL and Redis.
func (h *HealthHandler) Check(c *gin.Context) {
	healthy := true
	mysqlStatus := "ok"
	redisStatus := "ok"

	// Check MySQL
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.Ping() != nil {
		healthy = false
		mysqlStatus = "unreachable"
	}

	// Check Redis
	if err := h.rdb.Ping(c.Request.Context()).Err(); err != nil {
		healthy = false
		redisStatus = "unreachable"
	}

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"code":   errcode.Success,
		"status": map[bool]string{true: "ok", false: "degraded"}[healthy],
		"mysql":  mysqlStatus,
		"redis":  redisStatus,
	})
}
