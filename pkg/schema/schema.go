package schema

// SchemaKind differentiates schema categories.
type SchemaKind string

const (
	KindBotAPIMethod      SchemaKind = "bot_api_method"
	KindBotAPIType        SchemaKind = "bot_api_type"
	KindMiniAppComponent  SchemaKind = "mini_app_component"
	KindMiniAppType       SchemaKind = "mini_app_type"
	KindMiniAppEvent      SchemaKind = "mini_app_event"
)

// FieldType represents data types.
type FieldType string

const (
	TypeString  FieldType = "string"
	TypeInteger FieldType = "integer"
	TypeFloat   FieldType = "float"
	TypeBoolean FieldType = "boolean"
	TypeObject  FieldType = "object"
	TypeArray   FieldType = "array"
	TypeAny     FieldType = "any"
)

// FieldSpec defines one property or parameter within a schema.
type FieldSpec struct {
	Name          string    `json:"name"`
	Type          FieldType `json:"type"`
	Description   string    `json:"description"`
	Required      bool      `json:"required"`
	AllowedValues []string  `json:"allowed_values,omitempty"`
	MinLength     *int      `json:"min_length,omitempty"`
	MaxLength     *int      `json:"max_length,omitempty"`
	Minimum       *float64  `json:"minimum,omitempty"`
	Maximum       *float64  `json:"maximum,omitempty"`
}

// MethodSpec holds HTTP method details.
type MethodSpec struct {
	HTTPMethod string `json:"http_method"` // GET or POST
	Endpoint   string `json:"endpoint"`
	ReturnType string `json:"return_type"`
}

// Schema represents the complete structural contract for an API method or component.
type Schema struct {
	Name        string      `json:"name"`
	Kind        SchemaKind  `json:"kind"`
	Description string      `json:"description"`
	Method      *MethodSpec `json:"method,omitempty"`
	Fields      []FieldSpec `json:"fields"`
	DocURL      string      `json:"doc_url,omitempty"`
}

// GetField retrieves a field specification by name.
func (s *Schema) GetField(name string) (*FieldSpec, bool) {
	for i := range s.Fields {
		if s.Fields[i].Name == name {
			return &s.Fields[i], true
		}
	}
	return nil, false
}
