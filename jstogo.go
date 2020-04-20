// +build js, wasm

package utils

import (
	"fmt"
	"reflect"
	"syscall/js"
)

var zero = reflect.ValueOf(nil)

func GoValueOf(v js.Value, i interface{}) error {
	rv := reflect.ValueOf(i)
	if k := rv.Kind(); k != reflect.Ptr || rv.IsNil() {
		return &InvalidAssignmentError{Kind: k}
	}

	return recoverToGo(rv, v)
}

func recoverToGo(rv reflect.Value, jv js.Value) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = &InvalidAssignmentError{rec: rec}
		}
	}()

	_, err = toGo(rv, jv)
	return
}

func toGo(rv reflect.Value, jv js.Value) (reflect.Value, error) {
	if jv.IsNull() || jv.IsUndefined() {
		return zero, nil
	}

	k := rv.Kind()
	switch k {
	case reflect.Ptr:
		return goPointer(rv, jv)
	case reflect.Interface:
		if e := rv.Elem(); e != zero {
			return goInterface(rv, e, jv)
		}
	}

	switch t := jv.Type(); t {
	case js.TypeBoolean:
		return goBasic(rv, jv.Bool(), t)
	case js.TypeNumber:
		return goBasic(rv, jv.Float(), t)
	case js.TypeString:
		return goBasic(rv, jv.String(), t)
	case js.TypeObject:
		return goValue(rv, jv)
	default:
		return zero, &InvalidAssignmentError{Type: t, Kind: k}
	}
}

func goPointer(p reflect.Value, jv js.Value) (reflect.Value, error) {
	if p.IsNil() {
		p = reflect.New(p.Type().Elem())
	}

	v, err := toGo(p.Elem(), jv)
	if err != nil {
		return zero, err
	}
	if v != zero {
		p.Elem().Set(v)
	}
	return p, nil
}

func goInterface(i, e reflect.Value, jv js.Value) (reflect.Value, error) {
	v, err := toGo(e, jv)
	if err != nil {
		return zero, err
	}
	if v != zero {
		i.Set(v)
	}
	return i, nil
}

func goBasic(b reflect.Value, i interface{}, t js.Type) (val reflect.Value, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = &InvalidAssignmentError{Type: t, Kind: b.Kind()}
		}
	}()

	v := reflect.ValueOf(i)
	val = v.Convert(b.Type())
	return
}

func goValue(rv reflect.Value, jv js.Value) (reflect.Value, error) {
	switch k := rv.Kind(); k {
	case reflect.Struct:
		return goStruct(rv, jv)
	case reflect.Map:
		return goMap(rv, jv)
	case reflect.Slice:
		return goSlice(rv, jv)
	default:
		return zero, &InvalidAssignmentError{Type: jv.Type(), Kind: k}
	}
}

func goStruct(s reflect.Value, val js.Value) (reflect.Value, error) {
	t := s.Type()
	s = reflect.New(t).Elem()
	n := s.NumField()
	for i := 0; i < n; i++ {
		if f := s.Field(i); f.CanInterface() {
			k := nameOf(t.Field(i))
			jf := val.Get(k)
			v, err := toGo(f, jf)
			if err != nil {
				return zero, err
			}
			if v == zero {
				continue
			}
			f.Set(v)
		}
	}
	return s, nil
}

func goMap(m reflect.Value, jv js.Value) (reflect.Value, error) {
	t := m.Type()
	keys := object.Call("keys", jv)
	n := keys.Length()
	if m.IsNil() {
		m = reflect.MakeMapWithSize(t, n)
	}
	kt := t.Key()
	vt := t.Elem()
	for i := 0; i < n; i++ {
		jk := keys.Index(i)
		k := reflect.New(kt).Elem()
		k, err := toGo(k, jk)
		if err != nil {
			return zero, err
		}
		if k == zero {
			continue
		}
		jv := jv.Get(jk.String())
		v := reflect.New(vt).Elem()
		v, err = toGo(v, jv)
		if err != nil {
			return zero, err
		}
		if v == zero {
			continue
		}
		m.SetMapIndex(k, v)
	}
	return m, nil
}

func goSlice(s reflect.Value, jv js.Value) (reflect.Value, error) {
	t := s.Type()
	n := jv.Length()
	if s.IsNil() {
		s = reflect.MakeSlice(t, 0, n)
	}
	et := t.Elem()
	for i := 0; i < n; i++ {
		e := reflect.New(et).Elem()
		je := jv.Index(i)
		e, err := toGo(e, je)
		if err != nil {
			return zero, err
		}
		if e == zero {
			continue
		}
		s = reflect.Append(s, e)
	}
	return s, nil
}

type InvalidAssignmentError struct {
	Type js.Type
	Kind reflect.Kind
	rec  interface{}
}

func (e *InvalidAssignmentError) Error() string {
	if e.rec != nil {
		return fmt.Sprintf("unexpected panic: %+v", e.rec)
	}
	if e.Type == js.TypeUndefined {
		return fmt.Sprintf("invalid assignment to go: %v must be a non-nil pointer", e.Kind)
	}
	return fmt.Sprintf("invalid assignment from js: %v to go: %v", e.Type, e.Kind)
}
