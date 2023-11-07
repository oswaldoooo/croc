package internal

import (
	"reflect"
	"sync"
)

var (
	core_map             sync.Map //core map for store type
	SERVICE_NOT_SET      = Str_Error("service not set")
	SERVICE_NOT_REGISTER = Str_Error("service not register")
	METHOD_NOT_SET       = Str_Error("method not set")
	METHOD_NOT_FIND      = Str_Error("method not find")
	ARGS_ERROR           = Str_Error("args error")
)

func Call(src map[string]any) (ans map[string]any, err error) {

	if _, ok := src["service"]; !ok {
		err = SERVICE_NOT_SET
	} else if _, ok := src["method"]; !ok {
		err = METHOD_NOT_SET
	}
	if err == nil {
		ans = make(map[string]any)
		srv, ok := core_map.Load(src["service"])
		if ok {
			delete(src, "service")
			var ans_ []any
			ans_, err = call(srv, src)
			//fmt.Println("call result", ans_, err)
			if err == nil && len(ans_) > 0 {
				//fill ans map
				ans["result"] = ans_
			}
		} else {
			err = SERVICE_NOT_REGISTER
		}
	}
	return
}
func call(src any, args map[string]any) (ans []any, err error) {
	stp := reflect.TypeOf(src)
	if stp.NumMethod() == 0 {
		err = METHOD_NOT_FIND
	} else {
		if me, ok := stp.MethodByName(args["method"].(string)); ok {
			delete(args, "method")
			//fmt.Println(len(args), me.Type.NumIn())
			// rawans := me.Func.Call(transferTValue(args))
			if len(args) != me.Type.NumIn()-1 {
				err = ARGS_ERROR
				return
			}
			argarr := make([]reflect.Value, me.Type.NumIn())
			argarr[0] = reflect.ValueOf(src)
			if len(args) >= 1 {
				transferTValue(args, argarr[1:])
			}
			rawans := me.Func.Call(argarr)
			//fmt.Println("ans number", len(rawans))
			if len(rawans) > 0 {
				ans = transferTAny(rawans)
			}
		} else {
			err = METHOD_NOT_FIND
		}
	}
	return
}

func transfer[T any](v map[string]T) []T {
	var ans []T
	if len(v) == 0 {
		return ans
	}
	ans = make([]T, 0, len(v))
	for _, ele := range v {
		ans = append(ans, ele)
	}
	return ans
}
func transferTValue[T any](v map[string]T, dst []reflect.Value) {
	// ans = make([]reflect.Value, 0, len(v))
	start := 0
	for _, ele := range v {
		dst[start] = reflect.ValueOf(ele)
		start++
	}
}
func transferTAny(v []reflect.Value) []any {
	var ans = make([]any, 0, len(v))
	for _, ele := range v {
		ans = append(ans, ele.Interface())
	}
	return ans
}
func Register(name string, v any) {
	//fmt.Println("reister type", name)
	core_map.Store(name, v)
}
