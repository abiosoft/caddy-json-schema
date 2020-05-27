package jsonschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	vsCodeConfigDirectory = "./.vscode"
	vsCodeConfigFile      = "settings.json"

	// default permissions
	dirPerm  os.FileMode = 0700
	filePerm os.FileMode = 0600
)

// M is a convenience wrapper for JSON object.
type M map[string]interface{}

func writeToFile(w schemaWriter) error {
	if err := w.Prepare(); err != nil {
		return err
	}

	return w.Write()
}

var _ schemaWriter = (*vscodeWriter)(nil)
var _ schemaWriter = (*basicWriter)(nil)

type schemaWriter interface {
	Prepare() error
	Write() error
}

type basicWriter struct{}

func (b basicWriter) Prepare() error { return nil }
func (b basicWriter) Write() error {
	return jsonToFile(rootSchema, config.File, filePerm)
}

type file struct {
	filename string
	perm     os.FileMode
	exists   bool
}

type vscodeWriter struct {
	dir, config, schema file
	configJSON          map[string]interface{}

	ignoreConfig bool
}

func (v *vscodeWriter) prepareDirectory() error {
	// check if directory exists, retain permission
	if stat, err := os.Stat(vsCodeConfigDirectory); err == nil {
		if !stat.IsDir() {
			return errors.New("a file named '.vscode' exists")
		}
		// retain directory permission
		v.dir.perm = stat.Mode()
		v.dir.exists = true
	} else {
		err := os.MkdirAll(vsCodeConfigDirectory, dirPerm)
		if err != nil {
			return err
		}
		v.dir.perm = dirPerm
	}
	return nil
}

func (v *vscodeWriter) prepareFiles() error {
	// check if schema file exists, retain permission
	v.schema.filename = filepath.Join(vsCodeConfigDirectory, "caddy_schema.json")
	perm, ok, err := permOrDefault(v.schema.filename)
	if err != nil {
		return err
	}
	v.schema.exists = ok
	v.schema.perm = perm

	// check if vscode config file exists, retain permission
	v.config.filename = filepath.Join(vsCodeConfigDirectory, vsCodeConfigFile)
	perm, ok, err = permOrDefault(v.config.filename)
	if err != nil {
		return err
	}
	v.config.exists = ok
	v.config.perm = perm

	return nil
}

func (v *vscodeWriter) Prepare() error {
	if err := v.prepareDirectory(); err != nil {
		return err
	}
	if err := v.prepareFiles(); err != nil {
		return err
	}

	if v.config.exists {
		if err := v.loadVsConfig(); err != nil {
			return err
		}
	} else {
		v.configJSON = map[string]interface{}{}
	}

	if err := v.setVsConfig(); err != nil {
		return err
	}

	return nil
}

func (v *vscodeWriter) loadVsConfig() error {
	b, err := ioutil.ReadFile(v.config.filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &v.configJSON)
}

func (v *vscodeWriter) setJSONConfig() (bool, error) {
	const key = "json.schemas"
	var schemas []interface{}
	if s := v.configJSON[key]; s != nil {
		if _, ok := s.([]interface{}); !ok {
			return false, errors.New("invalid vscode config, 'json.schemas' not a list")
		}
		schemas = s.([]interface{})
	}

	for _, schema := range schemas {
		if s, ok := schema.(map[string]interface{}); ok {
			if url, ok := s["url"]; ok && url == v.schema.filename {
				return false, nil
			}
		}
	}

	schemas = append(schemas, M{
		"fileMatch": []interface{}{"*caddy*.json"},
		"url":       v.schema.filename,
	})

	v.configJSON[key] = schemas
	return true, nil
}

func (v *vscodeWriter) setYAMLConfig() (bool, error) {
	const key = "yaml.schemas"
	var schemas map[string]interface{}
	if s := v.configJSON[key]; s != nil {
		if _, ok := s.(map[string]interface{}); !ok {
			return false, errors.New("invalid vscode config, 'yaml.schemas' not an object")
		}
		schemas = s.(map[string]interface{})
	}

	for key := range schemas {
		if key == v.schema.filename {
			return false, nil
		}
	}

	if schemas == nil {
		schemas = make(map[string]interface{})
	}
	schemas[v.schema.filename] = []interface{}{
		"*caddy*.yaml",
		"*caddy*.yml",
	}

	v.configJSON[key] = schemas
	return true, nil
}

func (v *vscodeWriter) setVsConfig() error {
	var err error
	edited := struct{ json, yaml bool }{}

	// json
	edited.json, err = v.setJSONConfig()
	if err != nil {
		return err
	}

	// yaml
	edited.yaml, err = v.setYAMLConfig()
	if err != nil {
		return err
	}

	if !edited.json && !edited.yaml {
		v.ignoreConfig = true
		log.Println("vscode config found, ignoring...")
	}
	return nil
}

func (v *vscodeWriter) Write() error {
	err := jsonToFile(rootSchema, v.schema.filename, v.schema.perm)
	if err != nil {
		return err
	}

	if !v.ignoreConfig {
		return jsonToFile(v.configJSON, v.config.filename, v.config.perm)
	}

	return nil
}

// jsonToFile writes JSON obj to file.
func jsonToFile(obj interface{}, filename string, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	indentSpace := ""
	for i := 0; i < config.Indent; i++ {
		indentSpace += " "
	}
	encoder.SetIndent("", indentSpace)

	// var err error
	err = encoder.Encode(obj)
	if err != nil {
		return err
	}

	log.Println(filename, "written.")
	return nil
}

func permOrDefault(filename string) (os.FileMode, bool, error) {
	if stat, err := os.Stat(filename); err == nil {
		if stat.IsDir() {
			return 0, false, fmt.Errorf("a directory named '%s' exists", filename)
		}
		return stat.Mode(), true, nil
	}
	return filePerm, false, nil
}
