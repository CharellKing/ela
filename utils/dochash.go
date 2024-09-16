package utils

import (
	"math"
	"reflect"
)

func SanitizeData(data interface{}) interface{} {
	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.Map:
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			v.SetMapIndex(key, reflect.ValueOf(SanitizeData(val.Interface())))
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			v.Index(i).Set(reflect.ValueOf(SanitizeData(v.Index(i).Interface())))
		}
	case reflect.Float32, reflect.Float64:
		if math.IsInf(v.Float(), 0) || math.IsNaN(v.Float()) {
			return nil
		}
	default:

	}
	return data
}
