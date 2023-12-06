package restapi

import (
	"a0feed/utils/info"
	"net/http"

	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
)

func (s *Service) routes() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(ginzap.RecoveryWithZap(s.logger.Desugar(), true))

	pprof.Register(r, "debug/pprof")

	r.GET("/health", func(c *gin.Context) {
		var resp = struct {
			Status string `json:"status" example:"ok"`
		}{
			Status: "ok",
		}
		c.JSON(http.StatusOK, resp)
	})

	r.GET("/version", func(c *gin.Context) {
		var reps = struct {
			Name        string `json:"name" example:"App name"`
			Release     string `json:"release" example:"0.0.0"`
			BuildNumber string `json:"build_number" example:"#123"`
			CommitHash  string `json:"commit_hash" example:"1n3342jfds"`
			BuildTime   string `json:"build_time" example:"2020-07-30_13:56:15"`
		}{
			Name:        info.AppName,
			Release:     info.Release,
			BuildNumber: info.BuildNumber,
			CommitHash:  info.CommitHash,
			BuildTime:   info.BuildTime,
		}
		c.JSON(http.StatusOK, reps)
	})

	return r
}
