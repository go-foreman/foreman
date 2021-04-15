package message

import (
	"encoding/json"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type Marshaller interface {
	Unmarshal(b []byte) (Object, error)
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

	gk := unstructured.GroupKind()

	if gk.Empty() {
		return nil, WithDecoderErr(errors.Errorf("error decoding received message, GroupKind is empty. %v", unstructured))
	}

	obj, err := j.knownTypes.NewObject(gk)

	if err != nil {
		return nil, WithDecoderErr(errors.Wrapf(err, "creating instance of object for %s", gk.String()))
	}

	//message.Payload now is map[string]interface{} and we are going to fill this data into needed type from KnownTypesRegistry

	decoderConf := mapstructure.DecoderConfig{
		Squash:  true,
		TagName: "json",
		Result:  obj,
	}

	decoder, err := mapstructure.NewDecoder(&decoderConf)

	if err != nil {
		return nil, WithDecoderErr(errors.WithStack(err))
	}

	if err := decoder.Decode(unstructured.Object); err != nil {
		return nil, WithDecoderErr(errors.Errorf("error decoding Unstructured %v to an original type %s", unstructured, gk.String()))
	}

	resObj, ok := obj.(Object)

	if !ok {
		return nil, WithDecoderErr(errors.Errorf("error converting obj %s into Object, it does not implement interface", gk.String()))
	}

	return resObj, nil
}

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

//var objectType = reflect.TypeOf((*Object)(nil)).Elem()

func(j jsonDecoder) setGroupKind(obj Object) error {
	if gk := obj.GroupKind(); gk.Empty() {
		gk, err := j.knownTypes.ObjectKind(obj)
		if err != nil {
			return WithDecoderErr(errors.Wrapf(err, "encoding %v", obj))
		}
		obj.SetGroupKind(gk)
	}

	//structVal := reflect.ValueOf(obj)
	//structType := structVal.Type().Elem()
	//
	//for i := 0; i < structType.NumField(); i++ {
	//	if currentField := structType.Field(i).Type; currentField.Kind() == reflect.Interface {
	//		if currentField.Implements(objectType) {
	//			next, ok := structVal.Elem().Field(i).Interface().(Object)
	//			if !ok {
	//				return WithDecoderErr(errors.Errorf("converting %s to Object interface", structType.String()))
	//			}
	//			return j.setGroupKind(next)
	//		}
	//	}
	//}

	return nil
}
