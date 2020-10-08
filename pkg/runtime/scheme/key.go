package scheme

import (
	"fmt"
	"reflect"
	"strings"
)

//KeyChoice is a type that was summoned from golang configurations world for giving you an ability to chose how to register endpoints, handlers for messages.
type KeyChoice func() string

//WithKey sets string key
func WithKey(key string) KeyChoice {
	return func() string {
		return key
	}
}

//WithStruct reads variable and returns "pkgPath.StructName" as a key.
//Use it on your own risk. If you pass another type - unimaginary things will happen :)
func WithStruct(structVar interface{}) KeyChoice {
	return func() string {
		return GetStructTypeKey(structVar)
	}
}

func GetStructTypeKey(structVar interface{}) string {
	t := reflect.TypeOf(structVar)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return strings.ToLower(fmt.Sprintf("%s.%s", t.PkgPath(), t.Name()))
}
