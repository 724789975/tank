package user_http

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	common_config "server_manager/config"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
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
			klog.CtxErrorf(c.Request.Context(), "[AUTH-MISSING-HEADER] user-channel header is missing %s", c.Request.URL.Path)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		userInfo := UserInfo{}
		if err := json.Unmarshal([]byte(userInfoRaw), &userInfo); err != nil {
			klog.CtxErrorf(c.Request.Context(), "[AUTH-INVALID-JSON] Invalid JSON in user-channel header %s", err.Error())
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if userInfo.UserId == "" {
			klog.CtxErrorf(c.Request.Context(), "[AUTH-INVALID-USERID] Invalid userId in user-channel header %s", userInfoRaw)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if userInfo.Exp == 0 {
			klog.CtxErrorf(c.Request.Context(), "[AUTH-INVALID-EXP] Invalid exp in user-channel header %s", userInfoRaw)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		now := time.Now().Unix()
		if now-offset >= userInfo.Exp {
			klog.CtxErrorf(c.Request.Context(), "[AUTH-TOKEN-EXPIRED] Token expired %s", userInfoRaw)
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
	router.Use(otelgin.Middleware(common_config.Get("server_mgr_rpc.service_name").(string)))
	router.Use(Logger())
	router.Use(klogRecovery())
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
		context.Next()

		rt := time.Since(start).Milliseconds()
		if path == "/healthz" || path == "/ready" {
			return
		}
		if rt > 150 {
			klog.CtxErrorf(context.Request.Context(), "[HTTP-SLOW-API] path = %s  , rt = %d , userId = %s , slow api", path, rt, context.GetString("userId"))
		} else {
			klog.CtxInfof(context.Request.Context(), "[HTTP-ACCESS] path = %s  , rt = %d , userId = %s", path, rt, context.GetString("userId"))
		}
	}
}
