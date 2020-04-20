// +build js, wasm

package utils

import (
	"reflect"
	"strings"
	"syscall/js"
)

var (
	null   = js.ValueOf(nil)
	object = js.Global().Get("Object")
	array  = js.Global().Get("Array")
)

func JsValueOf(i interface{}) js.Value {
	switch i.(type) {
	case nil, js.Value, js.Wrapper:
		return js.ValueOf(i)
	default:
		v := reflect.ValueOf(i)
		return toJs(v)
	}
}

func toJs(v reflect.Value) js.Value {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return jsPointerOrInterface(v)
	case reflect.Slice, reflect.Array:
		return jsSliceOrArray(v)
	case reflect.Map:
		return jsMap(v)
	case reflect.Struct:
		return jsStruct(v)
	default:
		return js.ValueOf(v.Interface())
	}
}

func jsPointerOrInterface(v reflect.Value) js.Value {
	if v.IsNil() {
		return null
	}
	return toJs(v.Elem())
}

func jsSliceOrArray(v reflect.Value) js.Value {
	if v.IsNil() {
		return null
	}
	a := array.New()
	for i := 0; i < v.Len(); i++ {
		e := v.Index(i)
		a.SetIndex(i, toJs(e))
	}
	return a
}

func jsMap(v reflect.Value) js.Value {
	if v.IsNil() {
		return null
	}
	m := object.New()
	for i := v.MapRange(); i.Next(); {
		k := i.Key().Interface().(string)
		m.Set(k, toJs(i.Value()))
	}
	return m
}

func jsStruct(v reflect.Value) js.Value {
	s := object.New()
	for i := 0; i < v.NumField(); i++ {
		if f := v.Field(i); f.CanInterface() {
			if isEmptyValue(f) && canOmitEmpty(v.Type().Field(i)) {
				continue
			} else {
				k := nameOf(v.Type().Field(i))
				s.Set(k, toJs(f))
			}
		}
	}
	return s
}

func nameOf(sf reflect.StructField) string {
	jstag := sf.Tag.Get("js")
	if jstag == "" {
		return sf.Name
	}
	tokens := strings.Split(jstag, ",")
	if len(tokens) == 1 {
		return jstag
	} else {
		jstag = tokens[0]
		return jstag
	}
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func canOmitEmpty(sf reflect.StructField) bool {
	jstag := sf.Tag.Get("js")
	if jstag == "" {
		return false
	}
	tokens := strings.Split(jstag, ",")
	if len(tokens) == 1 {
		return false
	} else {
		switch param := tokens[1]; param {
		case "omitempty":
			return true
		default:
			return false
		}
	}
}
