package user_http

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"server_manager/tracer"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/gin-gonic/gin"
)

type UserInfo struct {
	UserId  string `json:"userId" binding:"required"`
	Exp     int64  `json:"exp" binding:"required"`
	Version string `json:"ver"`
}

func AuthForClient(offset int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/healthz" || c.Request.URL.Path == "/ready" {
			c.Next()
			return
		}
		userInfoRaw := c.GetHeader("user-channel")
		if userInfoRaw == "" {
			klog.Error("[AUTH-MISSING-HEADER] user-channel header is missing ", c.Request.URL.Path)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		userInfo := UserInfo{}
		if err := json.Unmarshal([]byte(userInfoRaw), &userInfo); err != nil {
			klog.Error("[AUTH-INVALID-JSON] Invalid JSON in user-channel header")
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if userInfo.UserId == "" {
			klog.Error("[AUTH-INVALID-USERID] Invalid userId in user-channel header")
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if userInfo.Exp == 0 {
			klog.Error("[AUTH-INVALID-EXP] Invalid exp in user-channel header")
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		now := time.Now().Unix()
		if now-offset >= userInfo.Exp {
			klog.Error("[AUTH-TOKEN-EXPIRED] Token expired")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set("userId", userInfo.UserId)

		c.Next()
	}
}

// 面向客户端的路由
func GetClientRouter() *gin.Engine {
	router := gin.New()
	router.Use(Logger())
	router.Use(klogRecovery())
	// router.Use(otelgin.Middleware(common.GetServiceName()))
	router.Use(AuthForClient(10))
	router.Any("/healthz", healthz)
	router.Any("/ready", healthz)
	return router
}

func klogRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err interface{}) {
		klog.CtxErrorf(c.Request.Context(), "[PANIC-RECOVERY] panic %v", err)
		klog.CtxErrorf(c.Request.Context(), "[PANIC-STACK] stack trace %s", debug.Stack())

		c.AbortWithStatus(500)
	})
}
func healthz(c *gin.Context) {
	c.String(http.StatusOK, "%s", "OK")
}

func Logger() gin.HandlerFunc {
	return func(context *gin.Context) {
		start := time.Now()
		path := context.Request.URL.Path
		ctx, span := tracer.GetCtxSpan()
		defer span.End()
		context.Next()

		rt := time.Since(start).Milliseconds()
		if path == "/healthz" || path == "/ready" {
			return
		}
		if rt > 150 {
			klog.CtxErrorf(ctx, "[HTTP-SLOW-API] path = %s  , rt = %d , userId = %s , body = %v , slow api", path, rt, context.GetString("userId"))
		} else {

			klog.CtxInfof(ctx, "[HTTP-ACCESS] path = %s  , rt = %d , userId = %s , body = %v", path, rt, context.GetString("userId"))
		}
	}
}
