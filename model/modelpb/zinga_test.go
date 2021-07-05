package modelpb

import (
	"fmt"
	"testing"

	"github.com/kr/pretty"
	"google.golang.org/protobuf/encoding/protojson"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

func TestZing(t *testing.T) {
	event := &APMEvent{
		Transaction: &Transaction{
			Id:                  "foo",
			Type:                "bar",
			RepresentativeCount: 123,
			Marks: map[string]*TransactionMarks{
				"baz": {
					Marks: []*TransactionMark{{
						Key:   "qux",
						Value: 123.456,
					}},
				},
			},
		},
	}
	jsonopts := protojson.MarshalOptions{UseProtoNames: true}
	out, _ := jsonopts.Marshal(event)
	fmt.Println(string(out))
	pretty.Println(convertMessage(event.ProtoReflect()))
}

func convertMessage(message protoreflect.Message) map[string]interface{} {
	object := make(map[string]interface{})
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if field.IsList() {
			in := value.List()
			out := make([]interface{}, in.Len())
			for i := range out {
				out[i] = convertValue(field, in.Get(i))
			}
			object[field.TextName()] = out
			return true
		}
		if field.IsMap() {
			in := value.Map()
			out := make(map[interface{}]interface{})
			in.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
				out[key.Interface()] = convertValue(field.MapValue(), value)
				return true
			})
			object[field.TextName()] = out
			return true
		}
		object[field.TextName()] = convertValue(field, value)
		return true
	})
	return object
}

func convertValue(field protoreflect.FieldDescriptor, value protoreflect.Value) interface{} {
	if field.Kind() == protoreflect.MessageKind {
		return convertMessage(value.Message())
	}
	return value.Interface()
}
