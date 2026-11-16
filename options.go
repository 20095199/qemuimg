package qemuimg

import (
	"sort"
	"strings"
)

// Option 是传给 qemu-img 的 -o 或 --image-opts 的单个键值对。
type Option struct {
	// Key 是选项名。
	Key string
	// Value 是选项值。
	Value string
}

// Options 是有序的 qemu-img 选项集合。
//
// 直接使用 Go map 会生成不确定的命令行。保留插入顺序可使生成的命令稳定，
// 从而让日志、测试和问题排查过程可复现。
type Options struct {
	pairs []Option
}

// NewOptions 按顺序返回包含给定键值对的 Options。
//
// 输入切片会被复制，因此调用方之后对切片的修改不会影响返回值。
func NewOptions(pairs ...Option) *Options {
	return &Options{pairs: append([]Option(nil), pairs...)}
}

// OptionsFromMap 基于 m 构造 Options。
//
// 键会被排序，因此无论 Go 的 map 遍历顺序如何，生成的命令行都是确定的。
// nil map 会生成一个空的、非 nil 的 Options。
func OptionsFromMap(m map[string]string) *Options {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	o := &Options{pairs: make([]Option, 0, len(keys))}
	for _, key := range keys {
		o.pairs = append(o.pairs, Option{Key: key, Value: m[key]})
	}
	return o
}

// Set 原地更新已存在键的值，或追加一个新的键值对。
//
// 对已存在的键保留原位置，可在调用方构造后更新选项时维持输出确定性。
// 对 nil 的 *Options 调用 Set 是无操作，返回 nil。
func (o *Options) Set(key, value string) *Options {
	if o == nil {
		return nil
	}
	for i := range o.pairs {
		if o.pairs[i].Key == key {
			o.pairs[i].Value = value
			return o
		}
	}
	o.pairs = append(o.pairs, Option{Key: key, Value: value})
	return o
}

// Get 返回 key 对应的值以及该键是否存在。
func (o *Options) Get(key string) (string, bool) {
	if o == nil {
		return "", false
	}
	for _, pair := range o.pairs {
		if pair.Key == key {
			return pair.Value, true
		}
	}
	return "", false
}

// Has 报告 key 是否存在。
func (o *Options) Has(key string) bool {
	_, ok := o.Get(key)
	return ok
}

// Delete 在 key 存在时将其删除。
func (o *Options) Delete(key string) {
	if o == nil {
		return
	}
	for i := range o.pairs {
		if o.pairs[i].Key == key {
			o.pairs = append(o.pairs[:i], o.pairs[i+1:]...)
			return
		}
	}
}

// Len 返回已存储选项的数量。
func (o *Options) Len() int {
	if o == nil {
		return 0
	}
	return len(o.pairs)
}

// Clone 返回 o 的独立副本。
//
// Clone 对 nil 安全：克隆 nil 的 *Options 会返回 nil。副本可被修改而不影响
// 原对象，从而让命令构建逻辑可以补入推断值，而不会修改调用方持有的请求数据。
func (o *Options) Clone() *Options {
	if o == nil {
		return nil
	}
	return &Options{pairs: append([]Option(nil), o.pairs...)}
}

// String 按插入顺序把选项渲染为以逗号连接的 "key=value" 对。
// 对 nil 或空的 Options 返回空字符串。
//
// 键为空的键值对会被跳过；它们对 qemu-img 没有意义，否则会产生
// 诸如 "=value" 的畸形输出。
func (o *Options) String() string {
	if o == nil || len(o.pairs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(o.pairs))
	for _, pair := range o.pairs {
		if pair.Key == "" {
			continue
		}
		parts = append(parts, pair.Key+"="+pair.Value)
	}
	return strings.Join(parts, ",")
}

// Map 返回包含与 o 相同键值对的普通 map。
//
// 如果某个键在有序表示中出现多次，则最后一个值生效，与普通 map 赋值语义一致。
func (o *Options) Map() map[string]string {
	m := make(map[string]string)
	if o == nil {
		return m
	}
	for _, pair := range o.pairs {
		if pair.Key == "" {
			continue
		}
		m[pair.Key] = pair.Value
	}
	return m
}
