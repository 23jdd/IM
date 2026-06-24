package tcp

// Context 连接级键值存储，用于在处理链中保存与某连接相关的临时数据。
type Context struct {
	store map[string]any
}

// NewContext 创建一个空的连接上下文。
func NewContext() *Context {
	return &Context{store: make(map[string]any)}
}

// Set 设置键值。
func (c *Context) Set(key string, val any) {
	c.store[key] = val
}

// Get 读取键值，第二个返回值表示键是否存在。
func (c *Context) Get(key string) (any, bool) {
	val, ok := c.store[key]
	return val, ok
}

// Del 删除指定键。
func (c *Context) Del(key string) {
	delete(c.store, key)
}
