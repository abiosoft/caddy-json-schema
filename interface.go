package jsonschema

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// Interface is a Go type representing a Caddy module structure
// or property.
type Interface struct {
	// keep track of the current module
	Module string

	// properties
	Name   string
	Fields []Interface
	Type   string

	// array/map type
	Array bool
	Map   bool // map key is always string
	Nest  *Interface

	// Module loaders
	Loader     []string // list of modules
	LoaderKey  string   // inline_key
	LoaderType reflect.Type
}

func (f Interface) goPkg() string {
	typ := reflect.TypeOf(flatModuleMap[f.Module].Type)
	if typ == nil {
		return ""
	}

	// dereference pointer to get underlying type
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// only return godoc for public fields
	if isPublic(typ.Name()) {
		return typ.PkgPath() + "." + typ.Name()
	}

	return ""
}

// toSchema converts the Interface to JSON schema.
func (f Interface) toSchema() *Schema {
	var s = NewSchema()
	s.setType(f.Type)

	// if it's a module loader, construct a special case (sub)schema
	if len(f.Loader) > 0 {
		l := moduleLoaderSchemaBuilder{s: s, f: f}
		l.build()
		l.apply(s)
	}

	// struct fields
	for _, field := range f.Fields {
		s.Properties[field.Name] = field.toSchema()
	}

	// get arrays and maps
	for cs, outer, nest := s, &f, f.Nest; nest != nil; outer, nest = nest, nest.Nest {
		props := map[string]*Schema{}
		for _, field := range nest.Fields {
			props[field.Name] = field.toSchema()
		}
		if outer.Array {
			cs.setType("array")
			cs.ArrayItems = NewSchema()
			cs.ArrayItems.setType(nest.Type)
			cs.ArrayItems.Properties = props

			// nested schema
			cs = cs.ArrayItems
		}

		if outer.Map {
			cs.setType("object")
			cs.AdditionalProperties = NewSchema()
			cs.AdditionalProperties.Properties = props

			// nested schema
			cs = cs.AdditionalProperties
		}
	}

	// now we're certain of the type
	s.description = description(f.Name, s.Type, f.Module)
	s.markdownDescription = markdownDescription(f.Name, s.Type, f.Module)
	s.goPkg = f.goPkg()

	// set the description in case JSON api docs not available
	// e.g. third party modules (for now)
	s.Description = s.description + "\n" + godocLink(s.goPkg)
	s.MarkdownDescription = s.markdownDescription + "  \n" + markdownLink("godoc", godocLink(s.goPkg))
	if s.goPkg == "" {
		s.Description = s.description
		s.MarkdownDescription = s.markdownDescription
	}
	return s
}

// populate populates the Interface with the type of s.
func (f *Interface) populate(s interface{}) {
	v := reflect.ValueOf(s)

	elemVal := func() interface{} { return reflect.Zero(v.Type().Elem()).Interface() }

	switch v.Kind() {
	case reflect.Struct:
		f.populateStruct(v.Type())

	case reflect.Ptr:
		// discard the pointer, use the underlying type
		f.populate(elemVal())

	case reflect.Slice:
		f.Array = true
		f.Nest = &Interface{
			Module: f.Module,
			Name:   f.Name + ".nest",
		}
		f.Nest.populate(elemVal())

	case reflect.Map:
		f.Map = true
		f.Nest = &Interface{
			Module: f.Module,
			Name:   f.Name + ".nest",
		}
		f.Nest.populate(elemVal())

	default:
		f.populateStruct(v.Type())
	}

}

func (f *Interface) populateStruct(t reflect.Type) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		f.Type = t.Kind().String()
		return
	}

	// rootLoaders are special type of module loaders where
	// module loading happens on the struct directly but not the
	// struct fields. Currently only applies to caddyhttp.MatchNot
	rootLoader := t == reflect.TypeOf(caddyhttp.MatchNot{})

	publicFields := []reflect.StructField{}
	for _, ff := range allFields(t) {
		if isPublic(ff.Name) {
			publicFields = append(publicFields, ff)
		}

		jsonTag, ok := ff.Tag.Lookup("json")

		if _, ok := ff.Tag.Lookup("caddy"); !ok {
			rootLoader = false // discard if missing struct tag
		}

		if (!ok || jsonTag == "-") && !rootLoader {
			continue
		}

		field := Interface{
			Module: f.Module,
			Name:   strings.TrimSuffix(jsonTag, ",omitempty"),
		}

		caddyTag, ok := ff.Tag.Lookup("caddy")
		if !ok {
			// regular fields
			field.populate(reflect.Zero(ff.Type).Interface())
			f.Fields = append(f.Fields, field)
			continue
		}

		// module loader fields
		split := strings.Fields(caddyTag)
		namespace := split[0] // 1 is inline_key
		namespace = strings.TrimPrefix(namespace, "namespace=")

		if len(split) > 1 {
			field.LoaderKey = strings.TrimPrefix(split[1], "inline_key=")
		}

		field.Module = namespace // use namespace as module
		field.LoaderType = ff.Type

		for key := range moduleMap[namespace] {
			modulePath := key
			if namespace != "" {
				modulePath = namespace + "." + key
			}
			field.Loader = append(field.Loader, modulePath)
		}

		if rootLoader {
			// delegate loading to parent struct
			// discard all fields
			f.Fields = nil
			f.Loader = field.Loader
			f.LoaderType = field.LoaderType
			f.LoaderKey = field.LoaderKey
			return
		}

		f.Fields = append(f.Fields, field)
	}

	// structs maps to object in JSON
	f.Type = getType("object")

	// for structs with custom unmarshalling and no json tagged fields.
	// if there's only one public field, assume the type of the field.
	// TODO: decide if this is necessary or simply leave as any.
	if len(f.Fields) == 0 && len(publicFields) == 1 {
		tmp := Interface{}
		tmp.populate(reflect.Zero(publicFields[0].Type).Interface())
		f.Type = tmp.Type
	}

}

func isPublic(fieldName string) bool {
	if fieldName == "" {
		return false
	}

	c := rune(fieldName[0])
	return unicode.IsUpper(c) && unicode.IsLetter(c)
}

// allFields retrives all struct fields (including nested structs) for a type.
func allFields(t reflect.Type) []reflect.StructField {
	// dereference pointer
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// reject non-structs
	if t.Kind() != reflect.Struct {
		return nil
	}

	var fields []reflect.StructField

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		if f.Anonymous {
			fields = append(fields, allFields(f.Type)...)
		} else {
			fields = append(fields, f)
		}

	}
	return fields
}

// moduleLoaderSchemaBuilder ... naming things is hard.
type moduleLoaderSchemaBuilder struct {
	f Interface
	s *Schema
}

func (m *moduleLoaderSchemaBuilder) build() {
	if m.f.LoaderKey != "" {
		m.buildWithInlineKey()
		return
	}

	m.s = NewSchema()
	// when loader key is absent, we expect a map[string]module
	for _, l := range m.f.Loader {
		split := strings.Split(l, ".")
		name := split[len(split)-1]

		ref := NewSchema()
		ref.setRef(l)

		m.s.Properties[name] = ref
	}
}

func (m *moduleLoaderSchemaBuilder) buildWithInlineKey() {
	m.s = NewSchema()
	// when loader key is set, we expect a []module.
	// combine {if, then} to improve suggestions.
	names := []string{}
	for _, l := range m.f.Loader {
		sub := NewSchema()

		sif := NewSchema()
		{
			split := strings.Split(l, ".")
			name := split[len(split)-1]

			tmp := NewSchema()
			tmp.Const = name
			names = append(names, name)

			sif.Properties[m.f.LoaderKey] = tmp
			sub.If = sif
		}

		sthen := NewSchema()
		{
			sthen.setRef(l)
			sub.Then = sthen
		}

		m.s.AllOf = append(m.s.AllOf, sub)
	}

	// make loaderKey a required field with appropriate suggestions
	inline := NewSchema()
	{
		tmp := NewSchema()
		tmp.setType("string")
		tmp.Enum = names

		// No way to generate docs for this yet.
		// TODO: make it reflect the current handler.
		desc := "key to identify %s module.\n%s: string\nModule: %s"
		mdDesc := "key to identify `%s` module.  \n%s: `string`  \nModule: `%s`"
		tmp.Description = fmt.Sprintf(desc, m.f.Name, m.f.LoaderKey, m.f.Module)
		tmp.MarkdownDescription = fmt.Sprintf(mdDesc, m.f.Name, m.f.LoaderKey, m.f.Module)

		inline.Properties[m.f.LoaderKey] = tmp
		m.s.AllOf = append(m.s.AllOf, inline)
	}

	m.s.Required = []string{m.f.LoaderKey}
}

func (m *moduleLoaderSchemaBuilder) apply(parent *Schema) {
	// use an hack to determine the module loader type

	// generate schema
	var tmp Interface
	tmp.populate(reflect.Zero(m.f.LoaderType).Interface())
	loaderSchema := tmp.toSchema()

	// determine how nested the generated schema is
	var nest int
	for cs := loaderSchema; cs.ArrayItems != nil || cs.AdditionalProperties != nil; nest++ {
		if cs.ArrayItems != nil {
			cs = cs.ArrayItems
		} else if cs.AdditionalProperties != nil {
			cs = cs.AdditionalProperties
		}
	}

	// derive the structure of the module loader from the nesting
	// TODO: use a different approach to improve readability
	switch nest {
	case 1:
		// module
		*parent = *m.s
		parent.setType("object")
	case 2:
		if m.f.LoaderKey == "" {
			// map[string]module
			*parent = *m.s
			parent.setType("object")
		} else {
			// []module
			parent.setType("array")
			parent.ArrayItems = m.s
		}
	case 3:
		// []map[string]module
		parent.setType("array")
		parent.ArrayItems = m.s
	}

}
