package huma

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var validateTests = []struct {
	name  string
	typ   reflect.Type
	input any
	mode  ValidateMode
	errs  []string
	panic string
}{
	{
		name:  "bool success",
		typ:   reflect.TypeOf(true),
		input: true,
	},
	{
		name:  "expected bool",
		typ:   reflect.TypeOf(true),
		input: 1.23,
		errs:  []string{"expected boolean"},
	},
	{
		name:  "int success",
		typ:   reflect.TypeOf(0),
		input: 0,
	},
	{
		name:  "float64 success",
		typ:   reflect.TypeOf(0),
		input: float64(0),
	},
	{
		name:  "int64 success",
		typ:   reflect.TypeOf(0),
		input: int64(0),
	},
	{
		name:  "expected number int",
		typ:   reflect.TypeOf(0),
		input: "",
		errs:  []string{"expected number"},
	},
	{
		name:  "expected number float64",
		typ:   reflect.TypeOf(float64(0)),
		input: "",
		errs:  []string{"expected number"},
	},
	{
		name: "minimum success",
		typ: reflect.TypeOf(struct {
			Value int `json:"value" minimum:"1"`
		}{}),
		input: map[string]any{"value": 1},
	},
	{
		name: "minimum fail",
		typ: reflect.TypeOf(struct {
			Value int `json:"value" minimum:"1"`
		}{}),
		input: map[string]any{"value": 0},
		errs:  []string{"expected number >= 1"},
	},
	{
		name: "exclusive minimum success",
		typ: reflect.TypeOf(struct {
			Value int `json:"value" exclusiveMinimum:"1"`
		}{}),
		input: map[string]any{"value": 2},
	},
	{
		name: "exclusive minimum fail",
		typ: reflect.TypeOf(struct {
			Value int `json:"value" exclusiveMinimum:"1"`
		}{}),
		input: map[string]any{"value": 1},
		errs:  []string{"expected number > 1"},
	},
	{
		name: "maximum success",
		typ: reflect.TypeOf(struct {
			Value int `json:"value" maximum:"1"`
		}{}),
		input: map[string]any{"value": 1},
	},
	{
		name: "maximum fail",
		typ: reflect.TypeOf(struct {
			Value int `json:"value" maximum:"1"`
		}{}),
		input: map[string]any{"value": 2},
		errs:  []string{"expected number <= 1"},
	},
	{
		name: "exclusive maximum success",
		typ: reflect.TypeOf(struct {
			Value int `json:"value" exclusiveMaximum:"1"`
		}{}),
		input: map[string]any{"value": 0},
	},
	{
		name: "exclusive maximum fail",
		typ: reflect.TypeOf(struct {
			Value int `json:"value" exclusiveMaximum:"1"`
		}{}),
		input: map[string]any{"value": 1},
		errs:  []string{"expected number < 1"},
	},
	{
		name: "multiple of success",
		typ: reflect.TypeOf(struct {
			Value int `json:"value" multipleOf:"5"`
		}{}),
		input: map[string]any{"value": 10},
	},
	{
		name: "multiple of fail",
		typ: reflect.TypeOf(struct {
			Value int `json:"value" multipleOf:"5"`
		}{}),
		input: map[string]any{"value": 2},
		errs:  []string{"expected number to be a multiple of 5"},
	},
	{
		name:  "string success",
		typ:   reflect.TypeOf(""),
		input: "",
	},
	{
		name:  "expected string",
		typ:   reflect.TypeOf(""),
		input: 1,
		errs:  []string{"expected string"},
	},
	{
		name: "min length success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" minLength:"1"`
		}{}),
		input: map[string]any{"value": "a"},
	},
	{
		name: "min length fail",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" minLength:"1"`
		}{}),
		input: map[string]any{"value": ""},
		errs:  []string{"expected length >= 1"},
	},
	{
		name: "max length success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" maxLength:"1"`
		}{}),
		input: map[string]any{"value": "a"},
	},
	{
		name: "max length fail",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" maxLength:"1"`
		}{}),
		input: map[string]any{"value": "ab"},
		errs:  []string{"expected length <= 1"},
	},
	{
		name: "pattern success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" pattern:"^[a-z]+$"`
		}{}),
		input: map[string]any{"value": "a"},
	},
	{
		name: "pattern fail",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" pattern:"^[a-z]+$"`
		}{}),
		input: map[string]any{"value": "a1"},
		errs:  []string{"expected string to match pattern ^[a-z]+$"},
	},
	{
		name: "pattern invalid",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" pattern:"^[a-"`
		}{}),
		input: map[string]any{"value": "a1"},
		panic: "error parsing regexp",
	},
	{
		name: "datetime success",
		typ: reflect.TypeOf(struct {
			Value time.Time `json:"value"`
		}{}),
		input: map[string]any{"value": []byte("2020-03-07T22:22:06-08:00")},
	},
	{
		name: "datetime string success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"date-time"`
		}{}),
		input: map[string]any{"value": []byte("2020-03-07T22:22:06-08:00")},
	},
	{
		name: "expected datetime",
		typ: reflect.TypeOf(struct {
			Value time.Time `json:"value"`
		}{}),
		input: map[string]any{"value": "bad"},
		errs:  []string{"expected string to be RFC 3339 date-time"},
	},
	{
		name: "date success",
		typ: reflect.TypeOf(struct {
			Value time.Time `json:"value" format:"date"`
		}{}),
		input: map[string]any{"value": "2020-03-07"},
	},
	{
		name: "expected date",
		typ: reflect.TypeOf(struct {
			Value time.Time `json:"value" format:"date"`
		}{}),
		input: map[string]any{"value": "bad"},
		errs:  []string{"expected string to be RFC 3339 date"},
	},
	{
		name: "time success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"time"`
		}{}),
		input: map[string]any{"value": "22:22:06-08:00"},
	},
	{
		name: "expected time",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"time"`
		}{}),
		input: map[string]any{"value": "bad"},
		errs:  []string{"expected string to be RFC 3339 time"},
	},
	{
		name: "email success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"email"`
		}{}),
		input: map[string]any{"value": "alice@example.com"},
	},
	{
		name: "expected email",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"email"`
		}{}),
		input: map[string]any{"value": "alice"},
		errs:  []string{"expected string to be RFC 5322 email: mail: missing '@' or angle-addr"},
	},
	{
		name: "hostname success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"hostname"`
		}{}),
		input: map[string]any{"value": "example.com"},
	},
	{
		name: "expected hostname",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"hostname"`
		}{}),
		input: map[string]any{"value": "%$^"},
		errs:  []string{"expected string to be RFC 5890 hostname"},
	},
	{
		name: "idn-hostname success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"idn-hostname"`
		}{}),
		input: map[string]any{"value": "ëxample.com"},
	},
	// {
	// 	name: "expected idn-hostname",
	// 	typ: reflect.TypeOf(struct {
	// 		Value string `json:"value" format:"idn-hostname"`
	// 	}{}),
	// 	input: map[string]any{"value": "\\"},
	// 	errs:  []string{"expected string to be RFC 5890 hostname"},
	// },
	{
		name: "ipv4 success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"ipv4"`
		}{}),
		input: map[string]any{"value": "127.0.0.1"},
	},
	{
		name: "expected ipv4",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"ipv4"`
		}{}),
		input: map[string]any{"value": "1234"},
		errs:  []string{"expected string to be RFC 2673 ipv4"},
	},
	{
		name: "ipv6 success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"ipv6"`
		}{}),
		input: map[string]any{"value": "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
	},
	{
		name: "expected ipv6",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"ipv6"`
		}{}),
		input: map[string]any{"value": "1234"},
		errs:  []string{"expected string to be RFC 2373 ipv6"},
	},
	{
		name: "uri success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"uri"`
		}{}),
		input: map[string]any{"value": "http://example.com"},
	},
	{
		name: "expected uri",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"uri"`
		}{}),
		input: map[string]any{"value": ":"},
		errs:  []string{"expected string to be RFC 3986 uri: parse \":\": missing protocol scheme"},
	},
	{
		name: "uuid success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"uuid"`
		}{}),
		input: map[string]any{"value": "123e4567-e89b-12d3-a456-426655440000"},
	},
	{
		name: "expected uuid",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"uuid"`
		}{}),
		input: map[string]any{"value": "bad"},
		errs:  []string{"expected string to be RFC 4122 uuid: invalid UUID length: 3"},
	},
	{
		name: "uritemplate success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"uri-template"`
		}{}),
		input: map[string]any{"value": "/items/{item-id}/history"},
	},
	{
		name: "expected uritemplate bad url",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"uri-template"`
		}{}),
		input: map[string]any{"value": ":"},
		errs:  []string{"expected string to be RFC 3986 uri: parse \":\": missing protocol scheme"},
	},
	{
		name: "expected uritemplate",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"uri-template"`
		}{}),
		input: map[string]any{"value": "missing{"},
		errs:  []string{"expected string to be RFC 6570 uri-template"},
	},
	{
		name: "jsonpointer success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"json-pointer"`
		}{}),
		input: map[string]any{"value": "/foo/bar"},
	},
	{
		name: "expected jsonpointer",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"json-pointer"`
		}{}),
		input: map[string]any{"value": "bad"},
		errs:  []string{"expected string to be RFC 6901 json-pointer"},
	},
	{
		name: "rel jsonpointer success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"relative-json-pointer"`
		}{}),
		input: map[string]any{"value": "0"},
	},
	{
		name: "expected rel jsonpointer",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"relative-json-pointer"`
		}{}),
		input: map[string]any{"value": "/bad"},
		errs:  []string{"expected string to be RFC 6901 relative-json-pointer"},
	},
	{
		name: "regex success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"regex"`
		}{}),
		input: map[string]any{"value": "^[0-9a-f]+$"},
	},
	{
		name: "expected regex",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" format:"regex"`
		}{}),
		input: map[string]any{"value": "^[bad"},
		errs:  []string{"expected string to be regex: error parsing regexp: missing closing ]: `[bad`"},
	},
	{
		name: "base64 byte success",
		typ: reflect.TypeOf(struct {
			Value []byte `json:"value"`
		}{}),
		input: map[string]any{"value": []byte("ABCD")},
	},
	{
		name: "base64 string success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" encoding:"base64"`
		}{}),
		input: map[string]any{"value": "ABCD"},
	},
	{
		name: "base64 fail",
		typ: reflect.TypeOf(struct {
			Value []byte `json:"value"`
		}{}),
		input: map[string]any{"value": []byte("!")},
		errs:  []string{"expected string to be base64 encoded"},
	},
	{
		name:  "array success",
		typ:   reflect.TypeOf([]any{}),
		input: []any{1, 2, 3},
	},
	{
		name:  "expected array",
		typ:   reflect.TypeOf([]any{}),
		input: 1,
		errs:  []string{"expected array"},
	},
	{
		name: "min items success",
		typ: reflect.TypeOf(struct {
			Value []any `json:"value" minItems:"1"`
		}{}),
		input: map[string]any{"value": []any{1}},
	},
	{
		name: "expected min items",
		typ: reflect.TypeOf(struct {
			Value []any `json:"value" minItems:"1"`
		}{}),
		input: map[string]any{"value": []any{}},
		errs:  []string{"expected array length >= 1"},
	},
	{
		name: "max items success",
		typ: reflect.TypeOf(struct {
			Value []any `json:"value" maxItems:"1"`
		}{}),
		input: map[string]any{"value": []any{1}},
	},
	{
		name: "expected max items",
		typ: reflect.TypeOf(struct {
			Value []any `json:"value" maxItems:"1"`
		}{}),
		input: map[string]any{"value": []any{1, 2}},
		errs:  []string{"expected array length <= 1"},
	},
	{
		name: "unique success",
		typ: reflect.TypeOf(struct {
			Value []any `json:"value" uniqueItems:"true"`
		}{}),
		input: map[string]any{"value": []any{1, 2, 3, 4, 5}},
	},
	{
		name: "expected unique",
		typ: reflect.TypeOf(struct {
			Value []any `json:"value" uniqueItems:"true"`
		}{}),
		input: map[string]any{"value": []any{1, 2, 1, 3}},
		errs:  []string{"expected array items to be unique"},
	},
	{
		name:  "map success",
		typ:   reflect.TypeOf(map[string]int{}),
		input: map[string]any{"one": 1, "two": 2},
	},
	{
		name:  "expected map item",
		typ:   reflect.TypeOf(map[string]int{}),
		input: map[string]any{"one": 1, "two": true},
		errs:  []string{"expected number"},
	},
	{
		name: "map minProps success",
		typ: reflect.TypeOf(struct {
			Value map[string]int `json:"value" minProperties:"1"`
		}{}),
		input: map[string]any{
			"value": map[string]any{"one": 1},
		},
	},
	{
		name: "expected map minProps",
		typ: reflect.TypeOf(struct {
			Value map[string]int `json:"value" minProperties:"1"`
		}{}),
		input: map[string]any{
			"value": map[string]any{},
		},
		errs: []string{"expected object with at least 1 properties"},
	},
	{
		name: "map maxProps success",
		typ: reflect.TypeOf(struct {
			Value map[string]int `json:"value" maxProperties:"1"`
		}{}),
		input: map[string]any{
			"value": map[string]any{"one": 1},
		},
	},
	{
		name: "expected map maxProps",
		typ: reflect.TypeOf(struct {
			Value map[string]int `json:"value" maxProperties:"1"`
		}{}),
		input: map[string]any{
			"value": map[string]any{"one": 1, "two": 2},
		},
		errs: []string{"expected object with at most 1 properties"},
	},
	{
		name:  "object struct success",
		typ:   reflect.TypeOf(struct{}{}),
		input: map[string]any{},
	},
	{
		name:  "expected object",
		typ:   reflect.TypeOf(struct{}{}),
		input: true,
		errs:  []string{"expected object"},
	},
	{
		name: "object optional success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value,omitempty"`
		}{}),
		input: map[string]any{},
	},
	{
		name: "readOnly set success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" readOnly:"true"`
		}{}),
		mode:  ModeWriteToServer,
		input: map[string]any{"value": "whoops"},
	},
	{
		name: "readOnly missing success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" readOnly:"true"`
		}{}),
		mode:  ModeWriteToServer,
		input: map[string]any{},
	},
	{
		name: "readOnly missing fail",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" readOnly:"true"`
		}{}),
		mode:  ModeReadFromServer,
		input: map[string]any{},
		errs:  []string{"expected required property value to be present"},
	},
	{
		name: "writeOnly missing fail",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" writeOnly:"true"`
		}{}),
		mode:  ModeReadFromServer,
		input: map[string]any{"value": "should not be set"},
		errs:  []string{"write only property is non-zero"},
	},
	{
		name: "unexpected property",
		typ: reflect.TypeOf(struct {
			Value string `json:"value,omitempty"`
		}{}),
		input: map[string]any{"value2": "whoops"},
		errs:  []string{"unexpected property"},
	},
	{
		name: "nested success",
		typ: reflect.TypeOf(struct {
			Items []struct {
				Value string `json:"value"`
			} `json:"items"`
		}{}),
		input: map[string]any{"items": []any{map[string]any{"value": "hello"}}},
	},
	{
		name: "expected nested",
		typ: reflect.TypeOf(struct {
			Items []struct {
				Value string `json:"value"`
			} `json:"items"`
		}{}),
		input: map[string]any{"items": []any{map[string]any{}}},
		errs:  []string{"expected required property value to be present"},
	},
	{
		name: "enum success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" enum:"one,two"`
		}{}),
		input: map[string]any{"value": "one"},
	},
	{
		name: "expected enum",
		typ: reflect.TypeOf(struct {
			Value string `json:"value" enum:"one,two"`
		}{}),
		input: map[string]any{"value": "three"},
		errs:  []string{"expected value to be one of \"one, two\""},
	},
	{
		name: "optional success",
		typ: reflect.TypeOf(struct {
			Value string `json:"value,omitempty" minLength:"1"`
		}{}),
		input: map[string]any{},
	},
	{
		name: "optional fail",
		typ: reflect.TypeOf(struct {
			Value string `json:"value,omitempty" minLength:"1"`
		}{}),
		input: map[string]any{"value": ""},
		errs:  []string{"expected length >= 1"},
	},
}

func TestValidate(t *testing.T) {
	pb := NewPathBuffer([]byte(""), 0)
	res := &ValidateResult{}

	for _, test := range validateTests {
		t.Run(test.name, func(t *testing.T) {
			registry := NewMapRegistry("#/components/schemas/", DefaultSchemaNamer)

			var s *Schema
			if test.panic != "" {
				assert.Panics(t, func() {
					registry.Schema(test.typ, false, "TestInput")
				})
				return
			} else {
				s = registry.Schema(test.typ, false, "TestInput")
			}

			pb.Reset()
			res.Reset()

			Validate(registry, s, pb, test.mode, test.input, res)

			if len(test.errs) > 0 {
				errs := mapTo(res.Errors, func(e error) string {
					return e.(*ErrorDetail).Message
				})
				schemaJSON, _ := json.MarshalIndent(registry.Map(), "", "  ")
				for _, err := range test.errs {
					assert.Contains(t, errs, err, string(schemaJSON))
				}
			} else {
				assert.Empty(t, res.Errors)
			}
		})
	}
}

var BenchValidatePB *PathBuffer
var BenchValidateRes *ValidateResult

func BenchmarkValidate(b *testing.B) {
	pb := NewPathBuffer([]byte(""), 0)
	res := &ValidateResult{}
	BenchValidatePB = pb
	BenchValidateRes = res

	for _, test := range validateTests {
		if test.panic != "" || len(test.errs) > 0 {
			continue
		}

		b.Run(strings.TrimSuffix(test.name, " success"), func(b *testing.B) {
			registry := NewMapRegistry("#/components/schemas/", DefaultSchemaNamer)
			s := registry.Schema(test.typ, false, "TestInput")

			input := test.input
			if s.Type == TypeObject && s.Properties["value"] != nil {
				s = s.Properties["value"]
				input = input.(map[string]any)["value"]
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				pb.Reset()
				res.Reset()
				Validate(registry, s, pb, test.mode, input, res)
			}
		})
	}
}
