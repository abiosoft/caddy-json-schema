package jsonschema

import (
	"fmt"
	"strings"
)

// rootSchema is the root schema where all sub schemas are added to.
// This holds the root `Config` structure of Caddy JSON config.
var rootSchema = NewSchema()

// NewSchema creates a new Schema. It's primarily for the
// convenience of initiating struct maps.
func NewSchema() *Schema {
	return &Schema{
		Definitions: make(map[string]*Schema),
		Properties:  make(map[string]*Schema),
	}
}

// Schema is a structure for JSON schema.
// JSON encoding of a Schema gives a valid JSON schema.
// http://json-schema.org
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

	// internal use for docs generation
	goPkg               string
	description         string
	markdownDescription string
}

func godocLink(pkg string) string {
	if pkg == "" {
		return ""
	}

	// api json docs uses dot for typename
	if i := strings.LastIndex(pkg, "."); i >= 0 {
		pkg = pkg[:i] + "#" + pkg[i+1:]
	}
	return "https://pkg.go.dev/" + pkg
}

func markdownLink(title, link string) string {
	if link == "" {
		return ""
	}
	return fmt.Sprintf("[%s](%s)", title, link)
}

func description(typeName, fieldType, module string) string {
	if fieldType == "" {
		fieldType = "any"
	}
	info := fmt.Sprintf("%s: %s\nModule: %s", typeName, fieldType, module)
	if module == "" {
		info = fmt.Sprintf("%s: %s", typeName, fieldType)
	}
	return info
}

func markdownDescription(typeName, fieldType, module string) string {
	if fieldType == "" {
		fieldType = "any"
	}
	info := fmt.Sprintf("%s: `%s`  \nModule: `%s`", typeName, fieldType, module)
	if module == "" {
		info = fmt.Sprintf("%s: `%s`", typeName, fieldType)
	}
	return info
}

func getType(typ string) string {
	switch typ {
	case "bool":
		return "boolean"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return "number"
	case "slice":
		return "array"
	case "any", "":
		return ""
	case "string", "array": // no transformation needed
		return typ
	}

	return "object"
}

func (s *Schema) setType(typ string) {
	s.Type = getType(typ)
}

func (s *Schema) setRef(moduleID string) {
	s.Ref = "#/definitions/" + moduleID
}
