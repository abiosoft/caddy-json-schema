package jsonschema

// moduleMap is map of namespaces to namespace modules.
// It is used by module loaders to identify modules in namespace.
var moduleMap = map[string]Modules{}

// flatModule is a flat map of module namespace to module
// without nesting.
// It is used during schema generation.
var flatModuleMap = Modules{}

// Module is a basic information about a Caddy module.
type Module struct {
	Name      string
	Type      interface{}
	Interface Interface
}

// Modules is list of Module. Map is used for quicker access
// and ease of fetching module name.
type Modules map[string]Module

// rootDocAPIResp holds the value for the API response of the root
// config.
var rootDocAPIResp DocAPIResp

// flatCaddyDocMap is a flat map of module path to API response
// without nesting.
// It is used during schema generation to retrieve module docs.
var flatCaddyDocMap = map[string]*DocAPIResp{}

// DocStruct is the API response structure for a type.
type DocStruct struct {
	Type    string `json:"type,omitempty"`
	Package string `json:"type_name,omitempty"`
	Doc     string `json:"doc,omitempty"`

	Key     string     `json:"key,omitempty"`
	Value   *DocStruct `json:"value,omitempty"`
	Elems   *DocStruct `json:"elems,omitempty"`
	MapKeys struct {
		Type string `json:"type,omitempty"`
	} `json:"map_keys,omitempty"`

	Namespace string `json:"module_namespace,omitempty"`
	InlineKey string `json:"module_inline_key,omitempty"`

	StructFields []*DocStruct `json:"struct_fields,omitempty"`
}

// DocAPIResp is the API response for a namespace documentation request.
type DocAPIResp struct {
	Structure  *DocStruct   `json:"structure,omitempty"`
	Namespaces DocNamespace `json:"namespaces,omitempty"`
}

// DocNamespace is the API response structure for namespaces
type DocNamespace map[string][]struct {
	Name string `json:"name,omitempty"`
	Docs string `json:"docs,omitempty"`

	Structure *DocStruct `json:"-"`
}
