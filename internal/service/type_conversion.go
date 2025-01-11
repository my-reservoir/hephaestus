package service

import (
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	v1 "hephaestus/api/lua/v1"
)

func Any(a *anypb.Any) (interface{}, error) {
	var msg interface{}
	switch {
	case a.MessageIs(&wrapperspb.StringValue{}):
		strVal := &wrapperspb.StringValue{}
		if err := a.UnmarshalTo(strVal); err != nil {
			return nil, err
		}
		msg = strVal.Value
	case a.MessageIs(&wrapperspb.Int32Value{}):
		intVal := &wrapperspb.Int32Value{}
		if err := a.UnmarshalTo(intVal); err != nil {
			return nil, err
		}
		msg = intVal.Value
	case a.MessageIs(&wrapperspb.BoolValue{}):
		boolVal := &wrapperspb.BoolValue{}
		if err := a.UnmarshalTo(boolVal); err != nil {
			return nil, err
		}
		msg = boolVal.Value
	case a.MessageIs(&wrapperspb.FloatValue{}):
		floatVal := &wrapperspb.FloatValue{}
		if err := a.UnmarshalTo(floatVal); err != nil {
			return nil, err
		}
		msg = floatVal.Value
	case a.MessageIs(&wrapperspb.DoubleValue{}):
		floatVal := &wrapperspb.DoubleValue{}
		if err := a.UnmarshalTo(floatVal); err != nil {
			return nil, err
		}
		msg = floatVal.Value
	case a.MessageIs(&wrapperspb.Int64Value{}):
		intVal := &wrapperspb.Int64Value{}
		if err := a.UnmarshalTo(intVal); err != nil {
			return nil, err
		}
		msg = intVal.Value
	case a.MessageIs(&wrapperspb.UInt32Value{}):
		intVal := &wrapperspb.UInt32Value{}
		if err := a.UnmarshalTo(intVal); err != nil {
			return nil, err
		}
		msg = intVal.Value
	case a.MessageIs(&wrapperspb.UInt64Value{}):
		intVal := &wrapperspb.UInt64Value{}
		if err := a.UnmarshalTo(intVal); err != nil {
			return nil, err
		}
		msg = intVal.Value
	case a.MessageIs(&structpb.Struct{}):
		m := make(map[string]interface{})
		structVal, err := structpb.NewStruct(m)
		if err != nil {
			return nil, err
		}
		if err := a.UnmarshalTo(structVal); err != nil {
			return nil, err
		}
		msg = m
	default:
		return nil, v1.ErrorInvalidParam("unknown type in Any: %v", a.TypeUrl)
	}
	return msg, nil
}

func AsAny(arg interface{}) (msg *anypb.Any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err, _ = r.(error)
		}
	}()
	switch v := arg.(type) {
	case int:
		return anypb.New(wrapperspb.Int32(int32(v)))
	case int8:
		return anypb.New(wrapperspb.Int32(int32(v)))
	case int16:
		return anypb.New(wrapperspb.Int32(int32(v)))
	case int32:
		return anypb.New(wrapperspb.Int32(v))
	case uint:
		return anypb.New(wrapperspb.UInt32(uint32(v)))
	case uint8:
		return anypb.New(wrapperspb.UInt32(uint32(v)))
	case uint16:
		return anypb.New(wrapperspb.UInt32(uint32(v)))
	case uint32:
		return anypb.New(wrapperspb.UInt32(v))
	case int64:
		return anypb.New(wrapperspb.Int64(v))
	case uint64:
		return anypb.New(wrapperspb.UInt64(v))
	case bool:
		return anypb.New(wrapperspb.Bool(v))
	case string:
		return anypb.New(wrapperspb.String(v))
	case float32:
		return anypb.New(wrapperspb.Float(v))
	case float64:
		return anypb.New(wrapperspb.Double(v))
	case []byte:
		return anypb.New(wrapperspb.Bytes(v))
	case nil:
		return anypb.New(structpb.NewNullValue())
	default:
		return anypb.New(structpb.NewNullValue())
	}
}

func ConvertFromProto(proto []*anypb.Any) (args []interface{}, err error) {
	args = make([]interface{}, 0, len(proto))
	for _, a := range proto {
		var v interface{}
		if v, err = Any(a); err != nil {
			return
		} else {
			args = append(args, v)
		}
	}
	return
}

func ConvertArgsToProto(args ...interface{}) (retVal *v1.ScriptReturnedValues, err error) {
	val := make([]*anypb.Any, 0, len(args))
	for _, arg := range args {
		var m *anypb.Any
		if m, err = AsAny(arg); err != nil {
			return
		}
		val = append(val, m)
	}
	retVal = &v1.ScriptReturnedValues{Args: val}
	return
}
