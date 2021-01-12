package hacache

import (
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/proto"
)

// Encoder encoder of ha-cache
type Encoder interface {
	Encode(v interface{}) ([]byte, error)
	Decode(b []byte) (interface{}, error)
	NewValue() interface{}
}

// HaEncoder default encoder of ha-cache
// 对于 protobuf message 使用 protobuf 序列化，其他 struct 使用 msgpack
type HaEncoder struct {
	// 函数返回缓存中存的值类型，空值指针
	// 用于将缓存中的值进行反序列化
	NewValueFn func() interface{}
}

// NewEncoder return new ha-encoder
func NewEncoder(n func() interface{}) *HaEncoder {
	return &HaEncoder{
		NewValueFn: n,
	}
}

// Encode encode v to bytes
func (enc *HaEncoder) Encode(v interface{}) ([]byte, error) {
	switch msg := v.(type) {
	case proto.Message:
		return proto.Marshal(msg)
	default:
		return msgpack.Marshal(v)
	}
}

// Decode decode bytes to interface
func (enc *HaEncoder) Decode(b []byte) (interface{}, error) {
	v := enc.NewValue()
	switch msg := v.(type) {
	case proto.Message:
		err := proto.Unmarshal(b, msg)
		return msg, err
	case int64:
		err := msgpack.Unmarshal(b, &msg)
		return msg, err
	default:
		err := msgpack.Unmarshal(b, &msg)
		return msg, err
	}
}

// NewValue return new empty encoder value
func (enc *HaEncoder) NewValue() interface{} {
	return enc.NewValueFn()
}
