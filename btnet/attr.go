package btnet

type AttrKey[T any] struct {
	name string
}

func NewAttrKey[T any](name string) *AttrKey[T] {
	return &AttrKey[T]{name: name}
}

func (k *AttrKey[T]) Name() string {
	return k.name
}

func (k *AttrKey[T]) Get(c *Connection) (T, bool) {
	v, ok := c.GetProperty(k)
	if !ok {
		var zero T
		return zero, false
	}
	t, ok := v.(T)
	return t, ok
}

func (k *AttrKey[T]) Set(c *Connection, v T) {
	c.SetProperty(k, v)
}

func (k *AttrKey[T]) Remove(c *Connection) {
	c.RemoveProperty(k)
}

func (c *Connection) SetProperty(key any, value any) {
	c.attMu.Lock()
	defer c.attMu.Unlock()
	if c.attrs == nil {
		c.attrs = make(map[any]any)
	}
	c.attrs[key] = value
}

func (c *Connection) GetProperty(key any) (any, bool) {
	c.attMu.RLock()
	defer c.attMu.RUnlock()
	v, ok := c.attrs[key]
	return v, ok
}

func (c *Connection) RemoveProperty(key any) {
	c.attMu.Lock()
	defer c.attMu.Unlock()
	delete(c.attrs, key)
}
