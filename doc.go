package jsonschema

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/caddyserver/caddy/v2"
)

func loadConfigJSONDoc() error {
	b, err := fetchConfigDocJSON("")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &caddyDoc); err != nil {
		return err
	}
	return nil
}

func populateJSONDocModules() error {
	// website json doc has repeated modules
	// prevent cycle
	visited := map[string]struct{}{}

	// top level namespaces
	for _, namespace := range caddyDoc.Namespaces[""] {
		b, err := fetchDocJSON(namespace.Name)
		if err != nil {
			return err
		}
		var tmp CaddyDoc
		if err := json.Unmarshal(b, &tmp); err != nil {
			return err
		}

		flatCaddyDocMap[namespace.Name] = &tmp

		// sub namespaces
		for ns, list := range tmp.Namespaces {
			if ns == "" {
				// avoid top level
				continue
			}
			for _, m := range list {
				modulePath := m.Name
				if ns != "" {
					modulePath = ns + "." + m.Name
				}
				// check if visited
				if _, ok := visited[modulePath]; ok {
					continue
				}
				// mark visited
				visited[modulePath] = struct{}{}

				flatCaddyDocMap[modulePath] = nil
			}
		}

	}
	return nil
}

// fetchJSONModuleDocs fetches docs for all available modules.
func fetchJSONModuleDocs() error {
	for ns, doc := range flatCaddyDocMap {
		if doc != nil {
			continue
		}

		b, err := fetchDocJSON(ns)
		if err != nil {
			return err
		}
		var tmp CaddyDoc
		if err := json.Unmarshal(b, &tmp); err != nil {
			return err
		}

		flatCaddyDocMap[ns] = &tmp
	}

	return nil
}

func loadJSONDoc() error {
	if err := loadConfigJSONDoc(); err != nil {
		return err
	}

	if err := populateJSONDocModules(); err != nil {
		return err
	}

	if err := fetchJSONModuleDocs(); err != nil {
		return err
	}

	return nil
}

// DocStruct ...
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

// CaddyDoc ...
type CaddyDoc struct {
	Structure  *DocStruct   `json:"structure,omitempty"`
	Namespaces DocNamespace `json:"namespaces,omitempty"`
}

// DocNamespace ...
type DocNamespace map[string][]struct {
	Name string `json:"name,omitempty"`
	Docs string `json:"docs,omitempty"`

	Structure *DocStruct `json:"-"`
}

var caddyDoc CaddyDoc
var flatCaddyDocMap = map[string]*CaddyDoc{}

func cacheFile(namespace string) (string, error) {
	fileName := "docs.json"
	dir := filepath.Join(caddy.AppDataDir(), "json_schema", namespace)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

func fetchDocJSON(namespace string) ([]byte, error) {
	return fetchConfigDocJSON("apps/" + namespace)
}

func fetchConfigDocJSON(config string) ([]byte, error) {
	// try local cache first
	cache, err := cacheFile(config)
	if err == nil {
		b, err := ioutil.ReadFile(cache)
		if err == nil {
			return b, nil
		}
	}

	apiURL := "https://caddyserver.com/api/docs/config/" + config
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	// cache file
	defer ioutil.WriteFile(cache, b, 0600)

	return b, nil
}
