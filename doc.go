package jsonschema

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/caddyserver/caddy/v2"
)

func loadJSONDoc() error {
	// app
	b, err := fetchDocJSON("")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &caddyDoc); err != nil {
		return err
	}

	// top level namespaces
	visited := map[string]struct{}{}
	for i, namespace := range caddyDoc.Namespaces[""] {
		b, err := fetchDocJSON(namespace.Name)
		if err != nil {
			return err
		}
		var tmp CaddyDoc
		if err := json.Unmarshal(b, &tmp); err != nil {
			return err
		}
		namespace.Structure = tmp.Structure
		caddyDoc.Namespaces[""][i] = namespace

		for ns, list := range tmp.Namespaces {
			names := []string{}
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

				names = append(names, modulePath)
				if m.Structure == nil {
					m.Structure = &DocStruct{
						Doc: m.Docs,
					}
				}
				caddyDoc.Namespaces[ns] = append(caddyDoc.Namespaces[ns], m)
			}
			log.Println(names)
		}
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

func cacheFile(namespace string) (string, error) {
	fileName := "docs.json"
	dir := filepath.Join(caddy.AppDataDir(), "json_schema", namespace)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

func fetchDocJSON(namespace string) ([]byte, error) {
	// try local cache first
	cache, err := cacheFile(namespace)
	if err == nil {
		b, err := ioutil.ReadFile(cache)
		if err == nil {
			return b, nil
		}
	}

	apiURL := "https://caddyserver.com/api/docs/config/apps/" + namespace
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
