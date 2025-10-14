package util

import "sync"

type RWMap[k comparable, T any] struct {
	data map[k]T
	mutex sync.RWMutex
}

func NewRWMap[k comparable, T any]() *RWMap[k, T] {
	return &RWMap[k, T]{
		data: make(map[k]T),
	}
}

func (r *RWMap[k, T]) Get(key k) (T, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	val, ok := r.data[key]
	return val, ok
}

func (r *RWMap[k, T]) Set(key k, val T) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.data[key] = val
}

func (r *RWMap[k, T]) Delete(key k) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.data, key)
}