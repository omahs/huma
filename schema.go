package huma

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

// ErrSchemaInvalid is sent when there is a problem building the schema.
var ErrSchemaInvalid = errors.New("schema is invalid")

// JSON Schema type constants
const (
	TypeBoolean = "boolean"
	TypeInteger = "integer"
	TypeNumber  = "number"
	TypeString  = "string"
	TypeArray   = "array"
	TypeObject  = "object"
)

var (
	timeType = reflect.TypeOf(time.Time{})
	ipType   = reflect.TypeOf(net.IP{})
	urlType  = reflect.TypeOf(url.URL{})
)

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// Schema represents a JSON Schema compatible with OpenAPI 3.1. It is extensible
// with your own custom properties. It supports a subset of the full JSON Schema
// spec, designed specifically for use with Go structs and to enable fast zero
// or near-zero allocation happy-path validation for incoming requests.
type Schema struct {
	Type                 string             `yaml:"type,omitempty"`
	Title                string             `yaml:"title,omitempty"`
	Description          string             `yaml:"description,omitempty"`
	Ref                  string             `yaml:"$ref,omitempty"`
	Format               string             `yaml:"format,omitempty"`
	ContentEncoding      string             `yaml:"contentEncoding,omitempty"`
	Default              any                `yaml:"default,omitempty"`
	Examples             []any              `yaml:"examples,omitempty"`
	Items                *Schema            `yaml:"items,omitempty"`
	AdditionalProperties any                `yaml:"additionalProperties,omitempty"`
	Properties           map[string]*Schema `yaml:"properties,omitempty"`
	Enum                 []any              `yaml:"enum,omitempty"`
	Minimum              *float64           `yaml:"minimum,omitempty"`
	ExclusiveMinimum     *float64           `yaml:"exclusiveMinimum,omitempty"`
	Maximum              *float64           `yaml:"maximum,omitempty"`
	ExclusiveMaximum     *float64           `yaml:"exclusiveMaximum,omitempty"`
	MultipleOf           *float64           `yaml:"multipleOf,omitempty"`
	MinLength            *int               `yaml:"minLength,omitempty"`
	MaxLength            *int               `yaml:"maxLength,omitempty"`
	Pattern              string             `yaml:"pattern,omitempty"`
	MinItems             *int               `yaml:"minItems,omitempty"`
	MaxItems             *int               `yaml:"maxItems,omitempty"`
	UniqueItems          bool               `yaml:"uniqueItems,omitempty"`
	Required             []string           `yaml:"required,omitempty"`
	MinProperties        *int               `yaml:"minProperties,omitempty"`
	MaxProperties        *int               `yaml:"maxProperties,omitempty"`
	ReadOnly             bool               `yaml:"readOnly,omitempty"`
	WriteOnly            bool               `yaml:"writeOnly,omitempty"`
	Deprecated           bool               `yaml:"deprecated,omitempty"`
	Extensions           map[string]any     `yaml:",inline"`

	patternRe     *regexp.Regexp  `yaml:"-"`
	requiredMap   map[string]bool `yaml:"-"`
	propertyNames []string        `yaml:"-"`

	// Precomputed validation messages. These prevent allocations during
	// validation and are known at schema creation time.
	msgEnum             string            `yaml:"-"`
	msgMinimum          string            `yaml:"-"`
	msgExclusiveMinimum string            `yaml:"-"`
	msgMaximum          string            `yaml:"-"`
	msgExclusiveMaximum string            `yaml:"-"`
	msgMultipleOf       string            `yaml:"-"`
	msgMinLength        string            `yaml:"-"`
	msgMaxLength        string            `yaml:"-"`
	msgPattern          string            `yaml:"-"`
	msgMinItems         string            `yaml:"-"`
	msgMaxItems         string            `yaml:"-"`
	msgMinProperties    string            `yaml:"-"`
	msgMaxProperties    string            `yaml:"-"`
	msgRequired         map[string]string `yaml:"-"`
}

func (s *Schema) PrecomputeMessages() {
	s.msgEnum = "expected value to be one of \"" + strings.Join(mapTo(s.Enum, func(v any) string {
		return fmt.Sprintf("%v", v)
	}), ", ") + "\""
	if s.Minimum != nil {
		s.msgMinimum = fmt.Sprintf("expected number >= %v", *s.Minimum)
	}
	if s.ExclusiveMinimum != nil {
		s.msgExclusiveMinimum = fmt.Sprintf("expected number > %v", *s.ExclusiveMinimum)
	}
	if s.Maximum != nil {
		s.msgMaximum = fmt.Sprintf("expected number <= %v", *s.Maximum)
	}
	if s.ExclusiveMaximum != nil {
		s.msgExclusiveMaximum = fmt.Sprintf("expected number < %v", *s.ExclusiveMaximum)
	}
	if s.MultipleOf != nil {
		s.msgMultipleOf = fmt.Sprintf("expected number to be a multiple of %v", *s.MultipleOf)
	}
	if s.MinLength != nil {
		s.msgMinLength = fmt.Sprintf("expected length >= %d", *s.MinLength)
	}
	if s.MaxLength != nil {
		s.msgMaxLength = fmt.Sprintf("expected length <= %d", *s.MaxLength)
	}
	if s.Pattern != "" {
		s.patternRe = regexp.MustCompile(s.Pattern)
		s.msgPattern = "expected string to match pattern " + s.Pattern
	}
	if s.MinItems != nil {
		s.msgMinItems = fmt.Sprintf("expected array length >= %d", *s.MinItems)
	}
	if s.MaxItems != nil {
		s.msgMaxItems = fmt.Sprintf("expected array length <= %d", *s.MaxItems)
	}
	if s.MinProperties != nil {
		s.msgMinProperties = fmt.Sprintf("expected object with at least %d properties", *s.MinProperties)
	}
	if s.MaxProperties != nil {
		s.msgMaxProperties = fmt.Sprintf("expected object with at most %d properties", *s.MaxProperties)
	}

	if s.Required != nil {
		if s.msgRequired == nil {
			s.msgRequired = map[string]string{}
		}
		for _, name := range s.Required {
			s.msgRequired[name] = "expected required property " + name + " to be present"
		}
	}
}

func (s *Schema) MarshalJSON() ([]byte, error) {
	return yaml.MarshalWithOptions(s, yaml.JSON())
}

func boolTag(f reflect.StructField, tag string) bool {
	if v := f.Tag.Get(tag); v != "" {
		if v == "true" {
			return true
		} else if v == "false" {
			return false
		} else {
			panic("invalid bool tag '" + tag + "' for field '" + f.Name + "': " + v)
		}
	}
	return false
}

func intTag(f reflect.StructField, tag string) *int {
	if v := f.Tag.Get(tag); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return &i
		} else {
			panic("invalid int tag '" + tag + "' for field '" + f.Name + "': " + v + " (" + err.Error() + ")")
		}
	}
	return nil
}

func floatTag(f reflect.StructField, tag string) *float64 {
	if v := f.Tag.Get(tag); v != "" {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			return &i
		} else {
			panic("invalid float tag '" + tag + "' for field '" + f.Name + "': " + v + " (" + err.Error() + ")")
		}
	}
	return nil
}

func jsonTagValue(f reflect.StructField, t reflect.Type, value string) any {
	// Special case: strings don't need quotes.
	if t.Kind() == reflect.String {
		return value
	}

	// Special case: array of strings with comma-separated values and no quotes.
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.String && value[0] != '[' {
		values := []string{}
		for _, s := range strings.Split(value, ",") {
			values = append(values, strings.TrimSpace(s))
		}
		return values
	}

	var v any
	if err := json.Unmarshal([]byte(value), &v); err != nil {
		panic("invalid tag for field '" + f.Name + "': " + err.Error())
	}

	vv := reflect.ValueOf(v)
	tv := reflect.TypeOf(v)
	if v != nil && tv != t {
		if tv.Kind() == reflect.Slice {
			// Slices can't be cast due to the different layouts. Instead, we make a
			// new instance of the destination slice, and convert each value in
			// the original to the new type.
			tmp := reflect.MakeSlice(t, 0, vv.Len())
			for i := 0; i < vv.Len(); i++ {
				if !vv.Index(i).Elem().Type().ConvertibleTo(t.Elem()) {
					panic(fmt.Errorf("unable to convert %v to %v: %w", vv.Index(i).Interface(), t.Elem(), ErrSchemaInvalid))
				}

				tmp = reflect.Append(tmp, vv.Index(i).Elem().Convert(t.Elem()))
			}
			v = tmp.Interface()
		} else if !tv.ConvertibleTo(t) {
			panic(fmt.Errorf("unable to convert %v to %v: %w", tv, t, ErrSchemaInvalid))
		}

		v = reflect.ValueOf(v).Convert(t).Interface()
	}

	return v
}

// jsonTag returns a value of the schema's type for the given tag string.
// Uses JSON parsing if the schema is not a string.
func jsonTag(f reflect.StructField, name string, multi bool) any {
	t := f.Type
	if value := f.Tag.Get(name); value != "" {
		return jsonTagValue(f, t, value)
	}
	return nil
}

func SchemaFromField(registry Registry, parent reflect.Type, f reflect.StructField) *Schema {
	parentName := ""
	if parent != nil {
		parentName = parent.Name()
	}
	fs := registry.Schema(f.Type, true, parentName+f.Name+"Struct")
	if fs == nil {
		return fs
	}
	fs.Description = f.Tag.Get("doc")
	if fmt := f.Tag.Get("format"); fmt != "" {
		fs.Format = fmt
	}
	if enc := f.Tag.Get("encoding"); enc != "" {
		fs.ContentEncoding = enc
	}
	fs.Default = jsonTag(f, "default", false)

	if e := jsonTag(f, "example", false); e != nil {
		fs.Examples = []any{e}
	}

	if enum := f.Tag.Get("enum"); enum != "" {
		fType := f.Type
		if fs.Type == TypeArray {
			fType = fType.Elem()
		}
		enumValues := []any{}
		for _, e := range strings.Split(enum, ",") {
			enumValues = append(enumValues, jsonTagValue(f, fType, e))
		}
		if fs.Type == TypeArray {
			fs.Items.Enum = enumValues
		} else {
			fs.Enum = enumValues
		}
	}
	fs.Minimum = floatTag(f, "minimum")
	fs.ExclusiveMinimum = floatTag(f, "exclusiveMinimum")
	fs.Maximum = floatTag(f, "maximum")
	fs.ExclusiveMaximum = floatTag(f, "exclusiveMaximum")
	fs.MultipleOf = floatTag(f, "multipleOf")
	fs.MinLength = intTag(f, "minLength")
	fs.MaxLength = intTag(f, "maxLength")
	fs.Pattern = f.Tag.Get("pattern")
	fs.MinItems = intTag(f, "minItems")
	fs.MaxItems = intTag(f, "maxItems")
	fs.UniqueItems = boolTag(f, "uniqueItems")
	fs.MinProperties = intTag(f, "minProperties")
	fs.MaxProperties = intTag(f, "maxProperties")
	fs.ReadOnly = boolTag(f, "readOnly")
	fs.WriteOnly = boolTag(f, "writeOnly")
	fs.Deprecated = boolTag(f, "deprecated")
	fs.PrecomputeMessages()

	return fs
}

// fieldInfo stores information about a field, which may come from an
// embedded type. The `Parent` stores the field's direct parent.
type fieldInfo struct {
	Parent reflect.Type
	Field  reflect.StructField
}

// getFields performs a breadth-first search for all fields including embedded
// ones. It may return multiple fields with the same name, the first of which
// represents the outer-most declaration.
func getFields(typ reflect.Type) []fieldInfo {
	fields := make([]fieldInfo, 0, typ.NumField())
	embedded := []reflect.StructField{}

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}

		if f.Anonymous {
			embedded = append(embedded, f)
			continue
		}

		fields = append(fields, fieldInfo{typ, f})
	}

	for _, f := range embedded {
		newTyp := f.Type
		for newTyp.Kind() == reflect.Ptr {
			newTyp = newTyp.Elem()
		}
		if newTyp.Kind() == reflect.Struct {
			fields = append(fields, getFields(newTyp)...)
		}
	}

	return fields
}

func SchemaFromType(r Registry, t reflect.Type) *Schema {
	s := Schema{}
	t = deref(t)

	if t == ipType {
		// Special case: IP address.
		return &Schema{Type: TypeString, Format: "ipv4"}
	}

	minZero := 0.0
	switch t.Kind() {
	case reflect.Bool:
		s.Type = TypeBoolean
	case reflect.Int:
		s.Type = TypeInteger
		if bits.UintSize == 32 {
			s.Format = "int32"
		} else {
			s.Format = "int64"
		}
	case reflect.Int8, reflect.Int16, reflect.Int32:
		s.Type = TypeInteger
		s.Format = "int32"
	case reflect.Int64:
		s.Type = TypeInteger
		s.Format = "int64"
	case reflect.Uint:
		s.Type = TypeInteger
		if bits.UintSize == 32 {
			s.Format = "int32"
		} else {
			s.Format = "int64"
		}
		s.Minimum = &minZero
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		// Unsigned integers can't be negative.
		s.Type = TypeInteger
		s.Format = "int32"
		s.Minimum = &minZero
	case reflect.Uint64:
		// Unsigned integers can't be negative.
		s.Type = TypeInteger
		s.Format = "int64"
		s.Minimum = &minZero
	case reflect.Float32:
		s.Type = TypeNumber
		s.Format = "float"
	case reflect.Float64:
		s.Type = TypeNumber
		s.Format = "double"
	case reflect.String:
		s.Type = TypeString
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			// Special case: []byte will be serialized as a base64 string.
			s.Type = TypeString
			s.ContentEncoding = "base64"
		} else {
			s.Type = TypeArray
			s.Items = r.Schema(t.Elem(), true, t.Name()+"Item")

			if t.Kind() == reflect.Array {
				len := t.Len()
				s.MinItems = &len
				s.MaxItems = &len
			}
		}
	case reflect.Map:
		s.Type = TypeObject
		s.AdditionalProperties = r.Schema(t.Elem(), true, t.Name()+"Value")
	case reflect.Struct:
		// Handle special cases.
		switch t {
		case timeType:
			return &Schema{Type: TypeString, Format: "date-time"}
		case urlType:
			return &Schema{Type: TypeString, Format: "uri"}
		}

		required := []string{}
		requiredMap := map[string]bool{}
		propNames := []string{}
		props := map[string]*Schema{}
		for _, info := range getFields(t) {
			f := info.Field

			name := f.Name
			omit := false
			if j := f.Tag.Get("json"); j != "" {
				name = strings.Split(j, ",")[0]
				if strings.Contains(j, "omitempty") {
					omit = true
				}
			}
			if name == "-" {
				// This field is deliberately ignored.
				continue
			}
			if props[name] != nil {
				// This field was overridden by an ancestor type, so we
				// should ignore it.
				continue
			}

			fs := SchemaFromField(r, info.Parent, f)
			if fs != nil {
				props[name] = fs
				propNames = append(propNames, name)
				if !omit {
					required = append(required, name)
					requiredMap[name] = true
				}
			}
		}
		s.Type = TypeObject
		s.AdditionalProperties = false
		s.Properties = props
		s.propertyNames = propNames
		s.Required = required
		s.requiredMap = requiredMap
		s.PrecomputeMessages()
	case reflect.Interface:
		// Interfaces mean any object.
	default:
		return nil
	}

	return &s
}
