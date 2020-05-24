package jsonschema

// Module ...
type Module struct {
	Name string      `json:"name,omitempty"`
	Type interface{} `json:"-"`
	Field
}

// Modules ...
type Modules map[string]Module

var moduleMap = map[string]Modules{}
var flatModuleMap = Modules{}

// Schema ...
type Schema struct {
	Title               string `json:"title,omitempty"`
	Description         string `json:"description,omitempty"`
	MarkdownDescription string `json:"markdownDescription,omitempty"`
	Type                string `json:"type,omitempty"`
	Ref                 string `json:"$ref,omitempty"`

	ArrayItems           *Schema            `json:"items,omitempty"`
	Definitions          map[string]*Schema `json:"definitions,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Enum                 []string           `json:"enum,omitempty"`

	AdditionalItems bool      `json:"additionalItems,omitempty"`
	AllOf           []*Schema `json:"allOf,omitempty"`
	AnyOf           []*Schema `json:"anyOf,omitempty"`
	OneOf           []*Schema `json:"oneOf,omitempty"`

	If   *Schema `json:"if,omitempty"`
	Then *Schema `json:"then,omitempty"`
	Else *Schema `json:"else,omitempty"`

	Const string `json:"const,omitempty"`
}

func getType(typ string) string {
	switch typ {
	case "bool":
		return "boolean"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return "number"
	case "string":
		return "string"
	case "slice", "array":
		return "array"
	}

	return "object"
}

func (s *Schema) setType(typ string) {
	s.Type = getType(typ)
}

func (s *Schema) setRef(moduleID string) {
	s.Ref = "#/definitions/" + moduleID
}

var globalSchema = NewSchema()

// NewSchema ...
func NewSchema() *Schema {
	return &Schema{
		Definitions: make(map[string]*Schema),
		Properties:  make(map[string]*Schema),
	}
}
