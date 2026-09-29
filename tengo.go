package tengo

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	// MaxStringLen is the maximum byte-length for string value. Note this
	// limit applies to all compiler/VM instances in the process.
	MaxStringLen = 2147483647

	// MaxBytesLen is the maximum length for bytes value. Note this limit
	// applies to all compiler/VM instances in the process.
	MaxBytesLen = 2147483647
)

const (
	// GlobalsSize is the maximum number of global variables for a VM.
	GlobalsSize = 1024

	// StackSize is the maximum stack size for a VM.
	StackSize = 2048

	// MaxFrames is the maximum number of function frames for a VM.
	MaxFrames = 1024

	// SourceFileExtDefault is the default extension for source files.
	SourceFileExtDefault = ".tengo"
)

// CallableFunc is a function signature for the callable functions.
type CallableFunc = func(args ...Object) (ret Object, err error)

// CountObjects returns the number of objects that a given object o contains.
// For scalar value types, it will always be 1. For compound value types,
// this will include its elements and all of their elements recursively.
func CountObjects(o Object) (c int) {
	c = 1
	switch o := o.(type) {
	case *Array:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *ImmutableArray:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *Map:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *ImmutableMap:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *Error:
		c += CountObjects(o.Value)
	}
	return
}

// ToString will try to convert object o to string value.
func ToString(o Object) (v string, ok bool) {
	if o == UndefinedValue {
		return
	}
	ok = true
	if str, isStr := o.(*String); isStr {
		v = str.Value
	} else {
		v = o.String()
	}
	return
}

// ToInt will try to convert object o to int value.
func ToInt(o Object) (v int, ok bool) {
	switch o := o.(type) {
	case *Int:
		v = int(o.Value)
		ok = true
	case *Float:
		v = int(o.Value)
		ok = true
	case *Char:
		v = int(o.Value)
		ok = true
	case *Bool:
		if o == TrueValue {
			v = 1
		}
		ok = true
	case *String:
		c, err := strconv.ParseInt(o.Value, 10, 64)
		if err == nil {
			v = int(c)
			ok = true
		}
	}
	return
}

// ToInt64 will try to convert object o to int64 value.
func ToInt64(o Object) (v int64, ok bool) {
	switch o := o.(type) {
	case *Int:
		v = o.Value
		ok = true
	case *Float:
		v = int64(o.Value)
		ok = true
	case *Char:
		v = int64(o.Value)
		ok = true
	case *Bool:
		if o == TrueValue {
			v = 1
		}
		ok = true
	case *String:
		c, err := strconv.ParseInt(o.Value, 10, 64)
		if err == nil {
			v = c
			ok = true
		}
	}
	return
}

// ToFloat64 will try to convert object o to float64 value.
func ToFloat64(o Object) (v float64, ok bool) {
	switch o := o.(type) {
	case *Int:
		v = float64(o.Value)
		ok = true
	case *Float:
		v = o.Value
		ok = true
	case *String:
		c, err := strconv.ParseFloat(o.Value, 64)
		if err == nil {
			v = c
			ok = true
		}
	}
	return
}

// ToBool will try to convert object o to bool value.
func ToBool(o Object) (v bool, ok bool) {
	ok = true
	v = !o.IsFalsy()
	return
}

// ToRune will try to convert object o to rune value.
func ToRune(o Object) (v rune, ok bool) {
	switch o := o.(type) {
	case *Int:
		v = rune(o.Value)
		ok = true
	case *Char:
		v = o.Value
		ok = true
	}
	return
}

// ToByteSlice will try to convert object o to []byte value.
func ToByteSlice(o Object) (v []byte, ok bool) {
	switch o := o.(type) {
	case *Bytes:
		v = o.Value
		ok = true
	case *String:
		v = []byte(o.Value)
		ok = true
	}
	return
}

// ToTime will try to convert object o to time.Time value.
func ToTime(o Object) (v time.Time, ok bool) {
	switch o := o.(type) {
	case *Time:
		v = o.Value
		ok = true
	case *Int:
		v = time.Unix(o.Value, 0)
		ok = true
	}
	return
}

// ToInterface attempts to convert an object o to an interface{} value
func ToInterface(o Object) (res interface{}) {
	switch o := o.(type) {
	case *Int:
		res = o.Value
	case *String:
		res = o.Value
	case *Float:
		res = o.Value
	case *Bool:
		res = o == TrueValue
	case *Char:
		res = o.Value
	case *Bytes:
		res = o.Value
	case *Array:
		res = make([]interface{}, len(o.Value))
		for i, val := range o.Value {
			res.([]interface{})[i] = ToInterface(val)
		}
	case *ImmutableArray:
		res = make([]interface{}, len(o.Value))
		for i, val := range o.Value {
			res.([]interface{})[i] = ToInterface(val)
		}
	case *Map:
		res = make(map[string]interface{})
		for key, v := range o.Value {
			res.(map[string]interface{})[key] = ToInterface(v)
		}
	case *ImmutableMap:
		res = make(map[string]interface{})
		for key, v := range o.Value {
			res.(map[string]interface{})[key] = ToInterface(v)
		}
	case *Time:
		res = o.Value
	case *Error:
		res = errors.New(o.String())
	case *Undefined:
		res = nil
	case ToGoValuer:
		return o.ToGoValue()
	case Object:
		return o
	}
	return
}

// ToGoValuer lets a custom Object control its own representation when
// converted back to a plain Go value via ToInterface. Without this, a
// custom Object falls through to the generic "case Object: return o"
// behavior below and comes back completely unconverted, which is often
// not what's wanted for a type that started life as ordinary Go data.
//
// This is purely opt-in: a custom Object that doesn't implement
// ToGoValuer is unaffected and keeps today's passthrough behavior.
type ToGoValuer interface {
	ToGoValue() interface{}
}

// FromInterface will attempt to convert an interface{} v to a Tengo Object
func FromInterface(v interface{}) (Object, error) {
	switch v := v.(type) {
	case nil:
		return UndefinedValue, nil
	case string:
		if len(v) > MaxStringLen {
			return nil, ErrStringLimit
		}
		return &String{Value: v}, nil
	case int64:
		return &Int{Value: v}, nil
	case int:
		return &Int{Value: int64(v)}, nil
	case bool:
		if v {
			return TrueValue, nil
		}
		return FalseValue, nil
	case rune:
		return &Char{Value: v}, nil
	case byte:
		return &Char{Value: rune(v)}, nil
	case float64:
		return &Float{Value: v}, nil
	case []byte:
		if len(v) > MaxBytesLen {
			return nil, ErrBytesLimit
		}
		return &Bytes{Value: v}, nil
	case error:
		return &Error{Value: &String{Value: v.Error()}}, nil
	case map[string]Object:
		return &Map{Value: v}, nil
	case map[string]interface{}:
		kv := make(map[string]Object)
		for vk, vv := range v {
			vo, err := FromInterface(vv)
			if err != nil {
				return nil, err
			}
			kv[vk] = vo
		}
		return &Map{Value: kv}, nil
	case []Object:
		return &Array{Value: v}, nil
	case []interface{}:
		arr := make([]Object, len(v))
		for i, e := range v {
			vo, err := FromInterface(e)
			if err != nil {
				return nil, err
			}
			arr[i] = vo
		}
		return &Array{Value: arr}, nil
	case time.Time:
		return &Time{Value: v}, nil
	case Object:
		return v, nil
	case CallableFunc:
		return &UserFunction{Value: v}, nil
	}
	return fromReflectValue(v)
}

// fromReflectValue is FromInterface's fallback for Go values that
// aren't one of the concrete types handled directly by its type switch:
// named bool/int/float/string types, and arbitrary slices, arrays,
// maps, structs, and pointers built from any of those (or from types
// already handled above). It mirrors how encoding/json walks a value.
//
// Scalar kinds are normalized to the base Go type (string, int64,
// float64, bool) and passed back through FromInterface, so they still
// go through the same explicit cases (and limits like MaxStringLen) as
// a direct call would; composite kinds recurse element-wise, also
// through FromInterface, so nested custom types and tengo.Object values
// are handled correctly at any depth.
//
// A cyclic Go value (a struct/slice/map that references itself) will
// recurse indefinitely here, the same as encoding/json.Marshal would on
// the same input; this intentionally matches that behavior rather than
// paying for cycle detection on every call.
func fromReflectValue(v interface{}) (Object, error) {
	rv := reflect.ValueOf(v)

	switch rv.Kind() {
	case reflect.Invalid:
		return UndefinedValue, nil
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return UndefinedValue, nil
		}
		return FromInterface(rv.Elem().Interface())
	case reflect.Bool:
		return FromInterface(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return FromInterface(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64:
		return FromInterface(int64(rv.Uint()))
	case reflect.Float32, reflect.Float64:
		return FromInterface(rv.Float())
	case reflect.String:
		return FromInterface(rv.String())
	case reflect.Slice, reflect.Array:
		n := rv.Len()
		arr := make([]Object, n)
		for i := 0; i < n; i++ {
			elem, err := FromInterface(rv.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			arr[i] = elem
		}
		return &Array{Value: arr}, nil
	case reflect.Map:
		kv := make(map[string]Object, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			key := iter.Key()
			keyStr := key.String()
			if key.Kind() != reflect.String {
				keyStr = fmt.Sprint(key.Interface())
			}
			val, err := FromInterface(iter.Value().Interface())
			if err != nil {
				return nil, err
			}
			kv[keyStr] = val
		}
		return &Map{Value: kv}, nil
	case reflect.Struct:
		t := rv.Type()
		kv := make(map[string]Object, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" { // unexported field
				continue
			}

			name := field.Name
			if tag, ok := field.Tag.Lookup("json"); ok {
				tagName, _, _ := strings.Cut(tag, ",")
				if tagName == "-" {
					continue
				}
				if tagName != "" {
					name = tagName
				}
			}

			val, err := FromInterface(rv.Field(i).Interface())
			if err != nil {
				return nil, err
			}
			kv[name] = val
		}
		return &Map{Value: kv}, nil
	}

	return nil, fmt.Errorf("cannot convert to object: %T", v)
}
