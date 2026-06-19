package tcp

type Context struct {
	store map[string]any
}

func NewContext() *Context {
	return &Context{store: make(map[string]any)}
}
func (c *Context) Set(key string, val any) {
	c.store[key] = val
}
func (c *Context) Get(key string) (any, bool) {
	val, ok := c.store[key]
	return val, ok
}
