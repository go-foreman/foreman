package message

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type Marshaller interface {
	// Unmarshal decodes received bytes into original type that must be registered in scheme
	Unmarshal(b []byte) (Object, error)
	// Marshal encodes a type and automatically adds needed metadata to resulting bytes
	Marshal(obj Object) ([]byte, error)
}

func NewJsonMarshaller(knownTypes scheme.KnownTypesRegistry) Marshaller {
	return &jsonDecoder{knownTypes: knownTypes}
}

type DecoderErr struct {
	error
}

func WithDecoderErr(err error) error {
	return DecoderErr{err}
}

type jsonDecoder struct {
	knownTypes scheme.KnownTypesRegistry
}

func (j jsonDecoder) Unmarshal(b []byte) (Object, error) {
	unstructured := &Unstructured{}

	if err := unstructured.UnmarshalJSON(b); err != nil {
		return nil, WithDecoderErr(err)
	}

	obj, err := j.decode(unstructured)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return obj, nil
}

func (j jsonDecoder) decode(unstructured *Unstructured) (Object, error) {
	parentObj, err := j.decodeUnstructured(unstructured)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err := j.walkUnstructured(parentObj, unstructured); err != nil {
		return nil, errors.WithStack(err)
	}

	return parentObj, nil
}

func (j jsonDecoder) walkUnstructured(parentObj Object, unstructured *Unstructured) error {
	for key, value := range unstructured.Object {
		if v, ok := value.(*Unstructured); ok {
			nestedObj, err := j.decodeUnstructured(v)
			if err != nil {
				return errors.WithStack(err)
			}

			parentValue := reflect.ValueOf(parentObj).Elem()
			fieldKey := findKeyByTag(parentValue.Type(), key)

			if fieldKey == "" {
				fieldKey = key
			}

			field := parentValue.FieldByName(fieldKey)

			if field.CanSet() {
				field.Set(reflect.ValueOf(nestedObj))
			}

			return j.walkUnstructured(nestedObj, v)
		}
	}

	return nil
}

func findKeyByTag(t reflect.Type, key string) string {
	tagsMap := make(map[string]string)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if jsonSpec := strings.Split(field.Tag.Get("json"), ","); len(jsonSpec) > 0 {
			tagsMap[jsonSpec[0]] = field.Name
		}
	}

	return tagsMap[key]
}

func (j jsonDecoder) decodeUnstructured(unstructured *Unstructured) (Object, error) {
	nestedObj, err := j.loadObj(unstructured)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	decoderConf := mapstructure.DecoderConfig{
		Squash:     true,
		TagName:    "json",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(toTimeHookFunc()),
		Result:     &nestedObj,
	}

	decoder, err := mapstructure.NewDecoder(&decoderConf)

	if err != nil {
		return nil, errors.Wrapf(err, "creating decoder for %v", unstructured.GroupKind().String())
	}

	if err := decoder.Decode(unstructured.Object); err != nil {
		return nil, errors.Wrapf(err, "decoding unstructured %s. data: %v", unstructured.GroupKind().String(), unstructured.Object)
	}

	return nestedObj, nil
}

func (j jsonDecoder) loadObj(unstructured *Unstructured) (Object, error) {
	gk := unstructured.GroupKind()

	if gk.Empty() {
		return nil, errors.Errorf("GroupKing is empty in Unstructured. data: %v", unstructured)
	}

	obj, err := j.knownTypes.NewObject(gk)

	if err != nil {
		return nil, errors.Wrapf(err, "creating new obj for %s data: %v", gk.String(), unstructured)
	}

	return obj, nil
}

// Marshal marshals Object into json. it has side effect though, it will set GK if obj has empty GK
func (j jsonDecoder) Marshal(obj Object) ([]byte, error) {
	encodingTo := obj

	//todo what if obj has embedded obj with already filled GroupKind? this obj will be marshalled with wrong GK
	if err := j.setGroupKind(obj); err != nil {
		return nil, errors.Wrapf(err, "setting GK recursively for %v", obj)
	}

	encodedBytes, err := json.Marshal(encodingTo)

	if err != nil {
		return nil, WithDecoderErr(errors.Wrapf(err, "encoding obj %v, GK: %s", obj, encodingTo.GroupKind().String()))
	}

	return encodedBytes, nil
}

var objectType = reflect.TypeOf((*Object)(nil)).Elem()

func (j jsonDecoder) setGroupKind(obj Object) error {
	if gk := obj.GroupKind(); gk.Empty() {
		gk, err := j.knownTypes.ObjectKind(obj)
		if err != nil {
			return WithDecoderErr(errors.Wrapf(err, "encoding %v", obj))
		}
		obj.SetGroupKind(gk)
	}

	//we do not allow nested Object
	structVal := reflect.ValueOf(obj)
	structType := structVal.Type().Elem()

	for i := 0; i < structType.NumField(); i++ {
		if currentField := structType.Field(i).Type; currentField.Kind() == reflect.Interface {
			if currentField.Implements(objectType) {
				next, ok := structVal.Elem().Field(i).Interface().(Object)
				if !ok {
					return WithDecoderErr(errors.Errorf("converting %s to Object interface", structType.String()))
				}
				return j.setGroupKind(next)
			}
		}
	}

	return nil
}

func toTimeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return time.Parse(time.RFC3339, data.(string))
		case reflect.Float64:
			return time.Unix(0, int64(data.(float64))*int64(time.Millisecond)), nil
		case reflect.Int64:
			return time.Unix(0, data.(int64)*int64(time.Millisecond)), nil
		default:
			return data, nil
		}
	}
}
