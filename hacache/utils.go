package hacache

import (
	"errors"
	"reflect"
)

// call fn(args...)
func call(fn interface{}, args ...interface{}) ([]reflect.Value, error) {
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		return nil, errors.New("invalid func")
	}

	f := reflect.ValueOf(fn)
	numIn := f.Type().NumIn()
	if len(args) < numIn {
		return nil, errors.New("args length not match")
	}

	in := make([]reflect.Value, numIn)
	for idx, arg := range args[:numIn] {
		in[idx] = reflect.ValueOf(arg)
	}

	return f.Call(in), nil
}

// copy 拷贝值，返回拷贝后的指针
func copyVal(v interface{}) interface{} {
	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		return nil
	}

	copied := reflect.ValueOf(v).Elem()
	copiedPtr := reflect.New(copied.Type())
	copiedPtr.Elem().Set(copied)
	return copiedPtr.Interface()
}
