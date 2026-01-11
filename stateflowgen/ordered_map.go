package stateflowgen

// OrderedMap 有序 Map,保证按插入顺序遍历
type OrderedMap[K comparable, V any] struct {
	keys   []K
	values map[K]V
}

// NewOrderedMap 创建新的有序 Map
func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		keys:   make([]K, 0),
		values: make(map[K]V),
	}
}

// Set 设置键值对,如果键不存在则添加到末尾
func (om *OrderedMap[K, V]) Set(key K, value V) {
	if _, exists := om.values[key]; !exists {
		om.keys = append(om.keys, key)
	}
	om.values[key] = value
}

// Get 获取值
func (om *OrderedMap[K, V]) Get(key K) (V, bool) {
	val, ok := om.values[key]
	return val, ok
}

// Has 检查键是否存在
func (om *OrderedMap[K, V]) Has(key K) bool {
	_, ok := om.values[key]
	return ok
}

// Keys 返回所有键(按插入顺序)
func (om *OrderedMap[K, V]) Keys() []K {
	return om.keys
}

// Len 返回元素数量
func (om *OrderedMap[K, V]) Len() int {
	return len(om.keys)
}
