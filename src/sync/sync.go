package sync

import (
	stdsync "sync"
)

type Lock[T any] struct {
	lock stdsync.Mutex
	item T
}

func MakeLock[T any](item T) Lock[T] {
	return Lock[T]{item: item}
}

// Set reassigns the item protected by this lock.
func (locked *Lock[T]) Set(item T) {
	locked.lock.Lock()
	locked.item = item
	locked.lock.Unlock()
}

// Do0 performs a potentially-interior-mutating action on the item without any return value.
func (locked *Lock[T]) Do0(action func(item T)) {
	locked.lock.Lock()
	defer locked.lock.Unlock()
	action(locked.item)
}

// Do1 performs a potentially-interior-mutating action on the item, and returing 1 value.
func (locked *Lock[T]) Do1[R any](action func(item T) R) R {
	locked.lock.Lock()
	defer locked.lock.Unlock()
	return action(locked.item)
}

// Do3 performs a potentially-interior-mutating action on the item, and returing 3 values.
func (locked *Lock[T]) Do3[R1, R2, R3 any](action func(item T) (R1, R2, R3)) (R1, R2, R3) {
	locked.lock.Lock()
	defer locked.lock.Unlock()
	return action(locked.item)
}
