package service

import (
	"context"

	"github.com/gin-gonic/gin"
)

type IService interface {
	ListenAndServe(ctx context.Context)
	ginRoute(ctx *gin.Context)
	Close()
}