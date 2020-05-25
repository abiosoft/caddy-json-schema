package jsonschema

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func loadDoc() error {
	if err := loadRootDoc(); err != nil {
		return err
	}

	if err := fetchAllDocumentedModules(); err != nil {
		return err
	}

	if err := fetchAllModuleDocs(); err != nil {
		return err
	}

	return nil
}

// loadRootDoc loads the documentation for the root config structure.
func loadRootDoc() error {
	b, err := fetchConfigDoc("")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &rootDocAPIResp); err != nil {
		return err
	}
	return nil
}

// fetchAllDocumentedModules() fetches and populate flatCaddyDocMap
// with all available documented modules.
func fetchAllDocumentedModules() error {
	// website json doc has repeated modules
	// prevent cycle
	visited := map[string]struct{}{}

	// top level namespaces
	for _, namespace := range rootDocAPIResp.Namespaces[""] {
		b, err := fetchNamespaceDoc(namespace.Name)
		if err != nil {
			return err
		}
		var tmp DocAPIResp
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
func fetchAllModuleDocs() error {
	for ns, doc := range flatCaddyDocMap {
		if doc != nil {
			continue
		}

		b, err := fetchNamespaceDoc(ns)
		if err != nil {
			return err
		}
		var tmp DocAPIResp
		if err := json.Unmarshal(b, &tmp); err != nil {
			return err
		}

		flatCaddyDocMap[ns] = &tmp
	}

	return nil
}

// fetchNamespaceDoc fetches the JSON doc for namespace
func fetchNamespaceDoc(namespace string) ([]byte, error) {
	return fetchConfigDoc("apps/" + namespace)
}

// fetchConfigDoc fetchs the JSON doc for config. e.g. admin, logging
func fetchConfigDoc(config string) ([]byte, error) {
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
