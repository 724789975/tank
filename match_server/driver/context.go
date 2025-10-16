package driver

import "context"

type Context struct {
	context.Context
	value map[any]any
}

func WithContext(ctx context.Context) *Context {
	return &Context{Context: ctx, value: map[any]any{}}
}

func (x *Context) Value(k any) any {
	if v, ok := x.value[k]; ok {
		return v
	}
	return nil
}

func (x *Context) WithValue(k, v any) {
	x.value[k] = v
}
