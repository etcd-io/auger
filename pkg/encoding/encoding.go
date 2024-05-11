/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package encoding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	yaml "gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const (
	StorageBinaryMediaType = "application/vnd.kubernetes.storagebinary"
	ProtobufMediaType      = "application/vnd.kubernetes.protobuf"
	YamlMediaType          = "application/yaml"
	JsonMediaType          = "application/json"

	ProtobufShortname = "proto"
	YamlShortname     = "yaml"
	JsonShortname     = "json"
)

// See k8s.io/apimachinery/pkg/runtime/serializer/protobuf.go
var ProtoEncodingPrefix = []byte{0x6b, 0x38, 0x73, 0x00}

// ToMediaType maps 'out' flag values to corresponding mime types.
func ToMediaType(out string) (string, error) {
	switch out {
	case YamlShortname:
		return YamlMediaType, nil
	case JsonShortname:
		return JsonMediaType, nil
	case ProtobufShortname:
		return ProtobufMediaType, nil
	default:
		return "", fmt.Errorf("unrecognized 'out' flag value: %v", out)
	}
}

// DetectAndConvert first detects the media type of the in param data and then converts it from
// the kv store encoded data to the given output format using kubernetes' api machinery to
// perform the conversion.
func DetectAndConvert(codecs serializer.CodecFactory, outMediaType string, in []byte) ([]byte, *runtime.TypeMeta, error) {
	inMediaType, in, err := DetectAndExtract(in)
	if err != nil {
		return nil, nil, err
	}
	return Convert(codecs, inMediaType, outMediaType, in)
}

// Convert from kv store encoded data to the given output format using kubernetes' api machinery to
// perform the conversion.
func Convert(codecs serializer.CodecFactory, inMediaType, outMediaType string, in []byte) ([]byte, *runtime.TypeMeta, error) {
	if inMediaType == StorageBinaryMediaType && outMediaType == ProtobufMediaType {
		unknown, err := DecodeUnknown(in)
		if err != nil {
			return nil, nil, err
		}
		return unknown.Raw, &unknown.TypeMeta, nil
	}

	if inMediaType == ProtobufMediaType && outMediaType == StorageBinaryMediaType {
		return nil, nil, fmt.Errorf("unsupported conversion: protobuf to kubernetes binary storage representation")
	}

	typeMeta, err := decodeTypeMeta(inMediaType, in)
	if err != nil {
		return nil, nil, err
	}

	var encoded []byte
	if inMediaType == outMediaType {
		// Assumes that the stored version is "correct". Primarily a short cut to allow CRDs to work.
		encoded = in
		if outMediaType == JsonMediaType {
			encoded = append(encoded, '\n')
		}
	} else if inMediaType == JsonMediaType && outMediaType == YamlMediaType {
		val := map[string]interface{}{}
		if err := json.Unmarshal(in, &val); err != nil {
			return nil, fmt.Errorf("error decoding from %s: %s", inMediaType, err)
		}
		encoded, err = yaml.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("error encoding from %s: %s", outMediaType, err)
		}
	} else {
		inCodec, err := newCodec(codecs, typeMeta, inMediaType)
		if err != nil {
			return nil, nil, err
		}
		outCodec, err := newCodec(codecs, typeMeta, outMediaType)
		if err != nil {
			return nil, nil, err
		}

		obj, err := runtime.Decode(inCodec, in)
		if err != nil {
			return nil, nil, fmt.Errorf("error decoding from %s: %s", inMediaType, err)
		}

		encoded, err = runtime.Encode(outCodec, obj)
		if err != nil {
			return nil, nil, fmt.Errorf("error encoding to %s: %s", outMediaType, err)
		}
	}

	return encoded, typeMeta, nil
}

// DetectAndExtract searches the the start of either json of protobuf data, and, if found, returns the mime type and data.
func DetectAndExtract(in []byte) (string, []byte, error) {
	if pb, ok := tryFindProto(in); ok {
		return StorageBinaryMediaType, pb, nil
	}
	if rawJs, ok := tryFindJson(in); ok {
		js, err := rawJs.MarshalJSON()
		if err != nil {
			return "", nil, err
		}
		return JsonMediaType, js, nil
	}
	return "", nil, fmt.Errorf("error reading input, does not appear to contain valid JSON or binary data")
}

// TryFindProto searches for the 'k8s\0' prefix, and, if found, returns the data starting with the prefix.
func tryFindProto(in []byte) ([]byte, bool) {
	i := bytes.Index(in, ProtoEncodingPrefix)
	if i >= 0 && i < len(in) {
		return in[i:], true
	}
	return nil, false
}

const jsonStartChars = "{["

// TryFindJson searches for the start of a valid json substring, and, if found, returns the json.
func tryFindJson(in []byte) (*json.RawMessage, bool) {
	var js json.RawMessage

	i := bytes.IndexAny(in, jsonStartChars)
	for i >= 0 && i < len(in) {
		in = in[i:]
		if len(in) < 2 {
			break
		}
		err := json.Unmarshal(in, &js)
		if err == nil {
			return &js, true
		}
		in = in[1:]
		i = bytes.IndexAny(in, jsonStartChars)
	}
	return nil, false
}

// DecodeSummary decodes the TypeMeta, ContentEncoding and ContentType fields from the 'Unknown'
// protobuf envelope of the given storage data.
func DecodeSummary(inMediaType string, in []byte, out io.Writer) error {
	typeMeta, err := decodeTypeMeta(inMediaType, in)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "TypeMeta.APIVersion: %s\n", typeMeta.APIVersion)
	fmt.Fprintf(out, "TypeMeta.Kind: %s\n", typeMeta.Kind)
	return nil
}

// DecodeUnknown decodes the Unknown protobuf type from the given storage data.
func DecodeUnknown(in []byte) (*runtime.Unknown, error) {
	if len(in) < 4 {
		return nil, fmt.Errorf("input too short, expected 4 byte proto encoding prefix but got %v", in)
	}
	if !bytes.Equal(in[:4], ProtoEncodingPrefix) {
		return nil, fmt.Errorf("first 4 bytes %v, do not match proto encoding prefix of %v", in[:4], ProtoEncodingPrefix)
	}
	data := in[4:]

	unknown := &runtime.Unknown{}
	if err := unknown.Unmarshal(data); err != nil {
		return nil, err
	}
	return unknown, nil
}

// NewCodec creates a new kubernetes storage codec for encoding and decoding persisted data.
func newCodec(codecs serializer.CodecFactory, typeMeta *runtime.TypeMeta, mediaType string) (runtime.Codec, error) {
	// For api machinery purposes, we treat StorageBinaryMediaType as ProtobufMediaType
	if mediaType == StorageBinaryMediaType {
		mediaType = ProtobufMediaType
	}
	mediaTypes := codecs.SupportedMediaTypes()

	info, ok := runtime.SerializerInfoForMediaType(mediaTypes, mediaType)
	if !ok {
		if len(mediaTypes) == 0 {
			return nil, fmt.Errorf("no serializers registered for %v", mediaTypes)
		}
		info = mediaTypes[0]
	}

	gv, err := schema.ParseGroupVersion(typeMeta.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("unable to parse meta APIVersion '%s': %s", typeMeta.APIVersion, err)
	}
	encoder := codecs.EncoderForVersion(info.Serializer, gv)
	decoder := codecs.DecoderToVersion(info.Serializer, gv)
	codec := codecs.CodecForVersions(encoder, decoder, gv, gv)
	return codec, nil
}

// getTypeMeta gets the TypeMeta from the given data, either as JSON or Protobuf.
func decodeTypeMeta(inMediaType string, in []byte) (*runtime.TypeMeta, error) {
	switch inMediaType {
	case JsonMediaType:
		return typeMetaFromJson(in)
	case StorageBinaryMediaType:
		return typeMetaFromBinaryStorage(in)
	case YamlMediaType:
		return typeMetaFromYaml(in)
	default:
		return nil, fmt.Errorf("unsupported inMediaType %s", inMediaType)
	}
}

func typeMetaFromJson(in []byte) (*runtime.TypeMeta, error) {
	var meta runtime.TypeMeta
	if err := json.Unmarshal(in, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func typeMetaFromBinaryStorage(in []byte) (*runtime.TypeMeta, error) {
	unknown, err := DecodeUnknown(in)
	if err != nil {
		return nil, err
	}
	return &unknown.TypeMeta, nil
}

func typeMetaFromYaml(in []byte) (*runtime.TypeMeta, error) {
	var meta runtime.TypeMeta
	if err := yaml.Unmarshal(in, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}
