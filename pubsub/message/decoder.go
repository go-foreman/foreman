package message

import (
	"encoding/json"
	"github.com/go-foreman/foreman/pubsub/transport/pkg"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type Decoder interface {
	Decode(inPkg pkg.IncomingPkg) (*Message, error)
}

func NewJsonDecoder(knownTypes scheme.KnownTypesRegistry) Decoder {
	return &JsonEncoder{knownTypes: knownTypes}
}

type DecoderErr struct {
	error
}

func WithDecoderErr(err error) error {
	return DecoderErr{err}
}

type JsonEncoder struct {
	knownTypes scheme.KnownTypesRegistry
}

func (j JsonEncoder) Decode(inPkg pkg.IncomingPkg) (*Message, error) {
	var decoded Message

	err := json.Unmarshal(inPkg.Payload(), &decoded)

	if err != nil {
		return nil, WithDecoderErr(err)
	}

	//message.Payload now is map[string]interface{} and we are going to fill this data into needed type from KnownTypesRegistry
	//then in any handler user will be able to do myType, ok := msg.Payload.(*MyType)

	payload, err := j.knownTypes.LoadType(scheme.WithKey(decoded.Name))

	if err != nil {
		return nil, WithDecoderErr(errors.Wrapf(err, "error decoding pkg payload into message"))
	}

	//kek, err := json.Marshal(decoded.Payload)
	//
	//if err != nil {
	//	return nil, WithDecoderErr(err)
	//}
	//
	//if err := json.Unmarshal(kek, &payload); err != nil {
	//	return nil, WithDecoderErr(errors.Wrapf(err, "Error decoding data from message payload interface{} to an original type %s", decoded.Name))
	//}

	decoderConf := mapstructure.DecoderConfig{
		Squash:  true,
		TagName: "json",
		Result:  &payload,
	}

	decoder, err := mapstructure.NewDecoder(&decoderConf)

	if err != nil {
		return nil, WithDecoderErr(errors.WithStack(err))
	}

	if err := decoder.Decode(decoded.Payload); err != nil {
		return nil, WithDecoderErr(errors.Errorf("Error decoding data from message payload interface{} to an original type %s", decoded.Name))
	}

	decoded.Payload = payload
	decoded.OriginSource = inPkg.Origin()
	decoded.Headers = inPkg.Headers()
	decoded.ReceivedAt = inPkg.ReceivedAt()

	return &decoded, nil
}
