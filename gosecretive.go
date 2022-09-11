package gosecretive

import (
	"fmt"
	"reflect"
)

// OnValueFuncHandler is a callback function
// Input:
// 		fieldPath - the path to the field (e.g. /spec/field, or /spec/field[0])
// 		valueToScrub - the value to scrub
// Output: a pointer to the new value to set (read: the secret key), or nil to keep the original value
// Notes:
// 1. callback must return unique values (unique enough that it will not collide with scrubbed/non-scrubbed values)
// 2. callback must return nil if it does not want to change the value
// 3. callback must return a pointer to the new value to set. aka: the secret key
type OnValueFuncHandler func(fieldPath string, valueToScrub interface{}) *string

// DefaultOnValueFuncHandler Simply replace the value to scrub with "$ref-" and the field path
var DefaultOnValueFuncHandler = func(fieldPath string, valueToScrub interface{}) *string {
	if valueStr, ok := valueToScrub.(string); ok {
		if valueStr != "" {
			scrubbedData := "$ref-" + fieldPath
			return &scrubbedData
		}
	}
	return nil
}

// Scrub will scrub the given object. if not scrubFuncHandler is given, default to DefaultOnValueFuncHandler.
// returns the scrubbed object and a map of the scrubbed data
// e.g.:
// 	original := map[string]interface{}{
// 	    "field": "value",
// 	}
// 	scrubbed, secrets := Scrub(&original, nil)
// 	fmt.Println(scrubbed)
// 	>> {"field": "$ref-/field"}
// 	fmt.Println(secrets)
// 	>> {"$ref-/field": "value"}
func Scrub(obj interface{}, scrubFuncHandler OnValueFuncHandler) (interface{}, map[string]string) {
	if scrubFuncHandler == nil {
		scrubFuncHandler = DefaultOnValueFuncHandler
	}
	vObj := reflect.ValueOf(obj)
	scrubbedObj := reflect.New(vObj.Type()).Elem()
	secrets := make(map[string]string)
	travel("", /* fieldPath */
		scrubbedObj,
		vObj,
		secrets,
		scrubFuncHandler)
	return scrubbedObj.Interface(), secrets
}

// Restore will restore the given object scrubbed data given the secrets map.
// travels the given object and replaces all the scrubbed data with the original data existing on secrets map.
// e.g.:
// 	scrubbed := map[string]interface{}{
// 	    "field": "$ref-/field",
// 	}
//  secrets := map[string]string{
// 	    "$ref-/field": "value",
// 	}
// 	restored := Restore(scrubbed, secrets)
// 	fmt.Println(restored)
// 	>> {"field": "value"}
func Restore(obj interface{}, secrets map[string]string) interface{} {
	vObj := reflect.ValueOf(obj)
	scrubbedObj := reflect.New(vObj.Type()).Elem()

	// copy the secrets map to avoid changing the original map
	secretsCopy := make(map[string]string)
	for k, v := range secrets {
		secretsCopy[k] = v
	}

	travel("/", /* fieldPath */
		scrubbedObj,
		vObj,
		secretsCopy,
		func(fieldPath string, value interface{}) *string {
			if valueStr, ok := value.(string); ok {
				if newValue, ok := secrets[valueStr]; ok {
					return &newValue
				}
			}
			return nil
		})
	return scrubbedObj.Interface()
}

// travel will travel the given object and call the given callback on specific values
func travel(fieldPath string,
	travelValue reflect.Value,
	original reflect.Value,
	secrets map[string]string,
	callback OnValueFuncHandler) {

	switch original.Kind() {

	// value is a pointer. dereference it and travel
	case reflect.Ptr:
		v := original.Elem()
		if !v.IsValid() {
			return
		}
		travelValue.Set(reflect.New(v.Type()))
		travel(fieldPath, travelValue.Elem(), v, secrets, callback)

	// value is an interface. dereference it, make a copy and travel the new copy
	case reflect.Interface:
		v := original.Elem()
		copyValue := reflect.New(v.Type()).Elem()
		travel(fieldPath, copyValue, v, secrets, callback)
		travelValue.Set(copyValue)

	// value is a struct. travel each field
	case reflect.Struct:
		for i := 0; i < original.NumField(); i += 1 {
			fieldName := original.Type().Field(i).Name
			travel(fmt.Sprintf("%s/%s", fieldPath, fieldName), travelValue.Field(i), original.Field(i), secrets, callback)
		}

	// value is a slice. travel each element
	case reflect.Slice:
		if original.IsNil() {
			return
		}
		travelValue.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
		for i := 0; i < original.Len(); i += 1 {
			travel(fmt.Sprintf("%s[%d]", fieldPath, i), travelValue.Index(i), original.Index(i), secrets, callback)
		}

	// value is a map. travel each element
	case reflect.Map:
		if original.IsNil() {
			return
		}
		travelValue.Set(reflect.MakeMap(original.Type()))
		for _, key := range original.MapKeys() {
			v := original.MapIndex(key)
			copyValue := reflect.New(v.Type()).Elem()
			travel(fmt.Sprintf("%s/%s", fieldPath, key.String()), copyValue, v, secrets, callback)
			travelValue.SetMapIndex(key, copyValue)
		}

	// value is a string. call the callback
	case reflect.String:
		newContent := callback(fieldPath, original.Interface())

		// callback returns a new value to set (override)
		if newContent != nil && original.String() != *newContent {
			secrets[*newContent] = original.String()
			travelValue.SetString(*newContent)
		} else {
			travelValue.SetString(original.String())
		}

	default:
		travelValue.Set(original)
	}

}
