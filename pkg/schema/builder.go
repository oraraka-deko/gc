package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// SchemaBuilder helps define schemas fluently.
type SchemaBuilder struct {
	schema Schema
}

// NewSchema creates a new SchemaBuilder.
func NewSchema(name string, kind SchemaKind) *SchemaBuilder {
	return &SchemaBuilder{
		schema: Schema{
			Name:   name,
			Kind:   kind,
			Fields: make([]FieldSpec, 0),
		},
	}
}

func (b *SchemaBuilder) Description(desc string) *SchemaBuilder {
	b.schema.Description = desc
	return b
}

func (b *SchemaBuilder) DocURL(url string) *SchemaBuilder {
	b.schema.DocURL = url
	return b
}

func (b *SchemaBuilder) Method(httpMethod, endpoint, returnType string) *SchemaBuilder {
	b.schema.Method = &MethodSpec{
		HTTPMethod: httpMethod,
		Endpoint:   endpoint,
		ReturnType: returnType,
	}
	return b
}

func (b *SchemaBuilder) Field(field FieldSpec) *SchemaBuilder {
	b.schema.Fields = append(b.schema.Fields, field)
	return b
}

func (b *SchemaBuilder) Build() *Schema {
	cp := b.schema
	return &cp
}

// PayloadBuilder constructs and validates input payloads against a schema.
type PayloadBuilder struct {
	schema *Schema
	data   map[string]any
	errors []string
}

// NewPayloadBuilder starts a payload builder for a given schema.
func NewPayloadBuilder(s *Schema) *PayloadBuilder {
	return &PayloadBuilder{
		schema: s,
		data:   make(map[string]any),
		errors: make([]string, 0),
	}
}

// Set adds or replaces a field in the payload.
func (p *PayloadBuilder) Set(field string, value any) *PayloadBuilder {
	p.data[field] = value
	return p
}

// SetIf sets a field only if condition is true.
func (p *PayloadBuilder) SetIf(condition bool, field string, value any) *PayloadBuilder {
	if condition {
		p.data[field] = value
	}
	return p
}

// Validate checks all required fields, constraints and types.
func (p *PayloadBuilder) Validate() error {
	if p.schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	var validationErrs []string

	for _, f := range p.schema.Fields {
		val, exists := p.data[f.Name]
		if !exists || val == nil {
			if f.Required {
				validationErrs = append(validationErrs, fmt.Sprintf("required field %q is missing", f.Name))
			}
			continue
		}

		// Allowed values validation
		if len(f.AllowedValues) > 0 {
			strVal := fmt.Sprintf("%v", val)
			matched := false
			for _, allowed := range f.AllowedValues {
				if strVal == allowed {
					matched = true
					break
				}
			}
			if !matched {
				validationErrs = append(validationErrs, fmt.Sprintf("field %q has value %q, allowed: %v", f.Name, strVal, f.AllowedValues))
			}
		}

		// Length checks for strings
		if f.Type == TypeString {
			if strVal, ok := val.(string); ok {
				if f.MinLength != nil && len(strVal) < *f.MinLength {
					validationErrs = append(validationErrs, fmt.Sprintf("field %q length %d is less than min %d", f.Name, len(strVal), *f.MinLength))
				}
				if f.MaxLength != nil && len(strVal) > *f.MaxLength {
					validationErrs = append(validationErrs, fmt.Sprintf("field %q length %d exceeds max %d", f.Name, len(strVal), *f.MaxLength))
				}
			}
		}

		// Numeric bounds checks
		if f.Type == TypeInteger || f.Type == TypeFloat {
			var numVal float64
			validNum := true
			switch v := val.(type) {
			case int:
				numVal = float64(v)
			case int64:
				numVal = float64(v)
			case float64:
				numVal = v
			case float32:
				numVal = float64(v)
			default:
				validNum = false
			}
			if validNum {
				if f.Minimum != nil && numVal < *f.Minimum {
					validationErrs = append(validationErrs, fmt.Sprintf("field %q value %f is less than min %f", f.Name, numVal, *f.Minimum))
				}
				if f.Maximum != nil && numVal > *f.Maximum {
					validationErrs = append(validationErrs, fmt.Sprintf("field %q value %f exceeds max %f", f.Name, numVal, *f.Maximum))
				}
			}
		}
	}

	if len(validationErrs) > 0 {
		return fmt.Errorf("validation failed: %v", validationErrs)
	}
	return nil
}

// Build validates and returns the final map payload.
func (p *PayloadBuilder) Build() (map[string]any, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	result := make(map[string]any, len(p.data))
	for k, v := range p.data {
		result[k] = v
	}
	return result, nil
}

// MustBuild returns payload or panics if invalid.
func (p *PayloadBuilder) MustBuild() map[string]any {
	res, err := p.Build()
	if err != nil {
		panic(err)
	}
	return res
}

// ToJSONSchema generates standard JSON Schema (draft-07) map from Schema.
func (s *Schema) ToJSONSchema() map[string]any {
	properties := make(map[string]any)
	required := make([]string, 0)

	for _, f := range s.Fields {
		fieldMap := map[string]any{
			"type":        string(f.Type),
			"description": f.Description,
		}
		if len(f.AllowedValues) > 0 {
			fieldMap["enum"] = f.AllowedValues
		}
		if f.MinLength != nil {
			fieldMap["minLength"] = *f.MinLength
		}
		if f.MaxLength != nil {
			fieldMap["maxLength"] = *f.MaxLength
		}
		if f.Minimum != nil {
			fieldMap["minimum"] = *f.Minimum
		}
		if f.Maximum != nil {
			fieldMap["maximum"] = *f.Maximum
		}
		properties[f.Name] = fieldMap

		if f.Required {
			required = append(required, f.Name)
		}
	}

	res := map[string]any{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"title":       s.Name,
		"type":        "object",
		"description": s.Description,
		"properties":  properties,
	}
	if len(required) > 0 {
		res["required"] = required
	}
	return res
}

// ToJSONSchemaString returns JSON formatted string of the JSON schema.
func (s *Schema) ToJSONSchemaString() (string, error) {
	js := s.ToJSONSchema()
	b, err := json.MarshalIndent(js, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Helper function to create int pointer
func IntPtr(i int) *int {
	return &i
}

// Helper function to create float64 pointer
func FloatPtr(f float64) *float64 {
	return &f
}

// PayloadFromStruct maps a struct to payload map using json tags.
func PayloadFromStruct(v any) map[string]any {
	out := make(map[string]any)
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return out
	}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("json")
		if tag == "-" || tag == "" {
			continue
		}
		parts := reflect.ValueOf(tag).String()
		name := parts
		if idx := len(name); idx > 0 {
			var fieldName string
			for j, ch := range name {
				if ch == ',' {
					fieldName = name[:j]
					break
				}
			}
			if fieldName == "" {
				fieldName = name
			}
			fVal := val.Field(i)
			if fVal.IsValid() && !fVal.IsZero() {
				out[fieldName] = fVal.Interface()
			}
		}
	}
	return out
}
