package service

import "context"

type IService interface {
	ListenAndServe(ctx context.Context)
	Close()
}