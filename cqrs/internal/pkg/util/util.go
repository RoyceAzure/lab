package util

import "reflect"

// IsNil 檢查介面是否為 nil
// 注意：這個函數會同時檢查介面的型別和值
// 只有當兩者都為 nil 時，才會返回 true
func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}

	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}

	return false
}

// HasImplementation 檢查介面是否有具體實體值
func HasImplementation(i interface{}) bool {
	if i == nil {
		return false
	}
	return !reflect.ValueOf(i).IsZero()
}
