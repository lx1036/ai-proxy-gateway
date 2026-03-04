package protomarshal

import (
	"errors"
	"github.com/golang/protobuf/jsonpb"
	legacyproto "github.com/golang/protobuf/proto" // nolint: staticcheck
	"google.golang.org/protobuf/proto"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
	"strings"
)

func ApplyYAML(yml string, pb proto.Message) error {
	js, err := yaml.YAMLToJSON([]byte(yml))
	if err != nil {
		return err
	}
	return ApplyJSON(string(js), pb)
}

// ApplyJSON unmarshals a JSON string into a proto message.
func ApplyJSON(js string, pb proto.Message) error {
	reader := strings.NewReader(js)
	m := jsonpb.Unmarshaler{}
	if err := m.Unmarshal(reader, legacyproto.MessageV1(pb)); err != nil {
		klog.Errorf("Failed to decode proto: %q. Trying decode with AllowUnknownFields=true", err)
		m.AllowUnknownFields = true
		reader.Reset(js)
		return m.Unmarshal(reader, legacyproto.MessageV1(pb))
	}
	return nil
}

// ToYAML marshals a proto to canonical YAML
func ToYAML(msg proto.Message) (string, error) {
	js, err := ToJSON(msg)
	if err != nil {
		return "", err
	}
	yml, err := yaml.JSONToYAML([]byte(js))
	return string(yml), err
}

// ToJSON marshals a proto to canonical JSON
func ToJSON(msg proto.Message) (string, error) {
	return ToJSONWithIndent(msg, "")
}

// ToJSONWithIndent marshals a proto to canonical JSON with pretty printed string
func ToJSONWithIndent(msg proto.Message, indent string) (string, error) {
	return ToJSONWithOptions(msg, indent, false)
}

// ToJSONWithOptions marshals a proto to canonical JSON with options to indent and
// print enums' int values
func ToJSONWithOptions(msg proto.Message, indent string, enumsAsInts bool) (string, error) {
	if msg == nil {
		return "", errors.New("unexpected nil message")
	}

	// Marshal from proto to json bytes
	m := jsonpb.Marshaler{Indent: indent, EnumsAsInts: enumsAsInts}
	return m.MarshalToString(legacyproto.MessageV1(msg))
}
