package jsonschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/caddyserver/caddy/v2"
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
	if rootDocAPIResp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d from caddyserver.com", rootDocAPIResp.StatusCode)
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
	for _, namespace := range rootDocAPIResp.Result.Namespaces[""] {
		b, err := fetchNamespaceDoc(namespace.Name)
		if err != nil {
			fmt.Println("error fetching namespace", namespace.Name, ":", err)
			continue
		}
		var tmp DocAPIResp
		if err := json.Unmarshal(b, &tmp); err != nil {
			fmt.Println("error unmarshaling namespace", namespace.Name, ":", err)
			continue
		}
		if tmp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code %d from caddyserver.com", rootDocAPIResp.StatusCode)
		}

		flatCaddyDocMap[namespace.Name] = &tmp

		// sub namespaces
		for ns, list := range tmp.Result.Namespaces {
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
			fmt.Println("error fetching namespace", ns, ":", err)
			continue
		}
		var tmp DocAPIResp
		if err := json.Unmarshal(b, &tmp); err != nil {
			fmt.Println("error unmarshaling namespace", ns, ":", err)
			continue
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

	label := config
	if config == "" {
		label = "root config"
	}

	if err == errCacheDisabled {
		log.Println("discarding cache for", label+".")
	} else {
		log.Println("cached docs not found for", label+".")
	}
	log.Println("fetching", apiURL, "...")

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

	log.Println()
	return b, nil
}

var errCacheDisabled = errors.New("cache disabled")

// cacheFile returns the filesystem path to cached API doc
// for namespace
func cacheFile(namespace string) (string, error) {
	if config.DiscardCache {
		return "", errCacheDisabled
	}

	fileName := "docs.json"
	dir := filepath.Join(caddy.AppDataDir(), "json_schema", namespace)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}
