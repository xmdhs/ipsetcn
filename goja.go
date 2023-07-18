package main

import (
	"fmt"

	"github.com/dop251/goja"
)

func NewJsFunc(s string) (f func(any, bool) (string, bool), err error) {
	vm := goja.New()
	_, err = vm.RunString(s)
	if err != nil {
		return nil, fmt.Errorf("NewJsFunc: %w", err)
	}

	need, ok := goja.AssertFunction(vm.Get("need"))
	if !ok {
		panic("Not a function")
	}

	return func(a any, b bool) (string, bool) {
		res, err := need(goja.Undefined(), vm.ToValue(a), vm.ToValue(b))
		if err != nil {
			panic(err)
		}
		o := res.ToObject(vm)
		return o.Get("tag").ToString().String(), o.Get("need").ToBoolean()
	}, nil

}
