package wechat

import "reflect"

// like in_array in php
func inArray(needle interface{}, haystack interface{}) (exists bool) {
	exists = false
	if reflect.TypeOf(haystack).Kind() != reflect.Slice {
		return
	}

	array := reflect.ValueOf(haystack)
	for i := 0; i < array.Len(); i++ {
		if reflect.DeepEqual(needle, array.Index(i).Interface()) {
			exists = true
			return
		}
	}

	return
}