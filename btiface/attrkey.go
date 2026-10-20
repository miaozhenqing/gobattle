package btiface

type AttrKey[T any] struct {
	name string
}

func NewAttrKey[T any](name string) *AttrKey[T] {
	return &AttrKey[T]{name: name}
}

func (k *AttrKey[T]) Name() string {
	return k.name
}

func (k *AttrKey[T]) Get(c IConnection) (T, bool) {
	v, ok := c.GetProperty(k)
	if !ok {
		var zero T
		return zero, false
	}
	t, ok := v.(T)
	return t, ok
}

func (k *AttrKey[T]) Set(c IConnection, v T) {
	c.SetProperty(k, v)
}

func (k *AttrKey[T]) Remove(c IConnection) {
	c.RemoveProperty(k)
}
