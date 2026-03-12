// Package jsoncodec provides a Gin JSON codec that serializes response keys as camelCase.
package jsoncodec

import (
	"bytes"
	"encoding/json"
	"io"

	convertor "github.com/chengshicheng/json-key-convertor"
	"github.com/iancoleman/strcase"
	ginjson "github.com/gin-gonic/gin/codec/json"
)

const lowerCamelName = "lowercamel"

func init() {
	convertor.RegisterConvertFunc(lowerCamelName, strcase.ToLowerCamel)
}

// camelCodec implements gin's codec/json.Core.
// Marshal outputs JSON with camelCase keys; Unmarshal uses standard behavior (snake_case input).
type camelCodec struct{}

// NewCamelCodec returns a gin codec that marshals with camelCase keys and unmarshals normally.
func NewCamelCodec() ginjson.Core {
	return &camelCodec{}
}

func (c *camelCodec) Marshal(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return convertor.ConvertKey(raw, lowerCamelName)
}

func (c *camelCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (c *camelCodec) MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	raw, err := json.MarshalIndent(v, prefix, indent)
	if err != nil {
		return nil, err
	}
	return convertor.ConvertKey(raw, lowerCamelName)
}

func (c *camelCodec) NewEncoder(w io.Writer) ginjson.Encoder {
	return &camelEncoder{w: w}
}

func (c *camelCodec) NewDecoder(r io.Reader) ginjson.Decoder {
	return json.NewDecoder(r)
}

type camelEncoder struct {
	w          io.Writer
	escapeHTML bool
}

func (e *camelEncoder) SetEscapeHTML(on bool) {
	e.escapeHTML = on
}

func (e *camelEncoder) Encode(v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(e.escapeHTML)
	if err := enc.Encode(v); err != nil {
		return err
	}
	converted, err := convertor.ConvertKey(buf.Bytes(), lowerCamelName)
	if err != nil {
		return err
	}
	_, err = e.w.Write(converted)
	return err
}
