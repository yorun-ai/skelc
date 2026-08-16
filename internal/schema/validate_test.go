package schema

import (
	"strings"
	"testing"
)

func TestDecodeRejectsInvalidSchemaDocuments(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"unsupported format", `{"format":"other.schema","formatVersion":1,"domain":"demo","declarations":[]}`},
		{"unsupported format version", `{"format":"yorun.skel.schema","formatVersion":2,"domain":"demo","declarations":[]}`},
		{"missing declarations", `{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo"}`},
		{"unknown declaration type", `{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","declarations":[{"pub":true,"name":"User","type":"unknown","skelName":"demo.User","data":{"members":[]}}]}`},
		{"unknown config lifecycle", `{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","declarations":[{"pub":true,"name":"Runtime","type":"config","skelName":"demo.Runtime","data":{"lifecycle":"session","members":[]}}]}`},
		{"unknown auth mode", `{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","declarations":[{"pub":true,"name":"Users","type":"service","skelName":"demo.Users","service":{"audiences":[],"auth":"sometimes","methods":[]}}]}`},
		{"unknown type kind", `{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","declarations":[{"pub":true,"name":"User","type":"data","skelName":"demo.User","data":{"members":[{"name":"id","type":{"kind":"mystery"}}]}}]}`},
		{"unknown requirement mode", `{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","declarations":[{"pub":true,"name":"Users","type":"service","skelName":"demo.Users","service":{"audiences":[],"auth":"unset","require":{"mode":"maybe"},"methods":[]}}]}`},
		{"incomplete required collection", `{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","declarations":[{"pub":true,"name":"User","type":"data","skelName":"demo.User","data":{}}]}`},
		{"null member type", `{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","declarations":[{"pub":true,"name":"User","type":"data","skelName":"demo.User","data":{"members":[{"name":"id","type":null}]}}]}`},
		{"unrelated type fields", `{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","declarations":[{"pub":true,"name":"User","type":"data","skelName":"demo.User","data":{"members":[{"name":"id","type":{"kind":"scalar","name":"string","element":{"kind":"scalar","name":"string"}}}]}}]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Decode(strings.NewReader(test.json)); err == nil {
				t.Fatal("expected schema validation to fail")
			}
		})
	}
}
