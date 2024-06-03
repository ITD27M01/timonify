package format

import (
	"reflect"
	"strconv"
)

func QuoteStringsInStruct(s interface{}) {
	v := reflect.ValueOf(s)

	// If the value is a pointer, get the element it points to
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		if v.String() != "" && v.CanSet() {
			v.SetString(strconv.Quote(v.String()))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if field.Kind() == reflect.Ptr {
				field = field.Elem()
			}
			if field.CanAddr() && field.CanInterface() {
				QuoteStringsInStruct(field.Addr().Interface())
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			if elem.CanAddr() && elem.CanInterface() {
				QuoteStringsInStruct(elem.Addr().Interface())
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			if value.Kind() == reflect.Ptr {
				value = value.Elem()
			}
			newValue := reflect.New(value.Type()).Elem()
			newValue.Set(value)
			QuoteStringsInStruct(newValue.Addr().Interface())
			v.SetMapIndex(key, newValue)
		}
	}
}
