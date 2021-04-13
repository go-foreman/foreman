package message

import (
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type Decoder interface {
	Decode(b []byte) (Object, error)
}

func NewJsonDecoder(knownTypes scheme.KnownTypesRegistry) Decoder {
	return &JsonDecoder{knownTypes: knownTypes}
}

type DecoderErr struct {
	error
}

func WithDecoderErr(err error) error {
	return DecoderErr{err}
}

type JsonDecoder struct {
	knownTypes scheme.KnownTypesRegistry
}

func (j JsonDecoder) Decode(b []byte) (Object, error) {
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
