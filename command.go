package jsonschema

import (
	"flag"

	"github.com/caddyserver/caddy/v2"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"
)

var config = struct {
	File   string
	VsCode bool
	Indent int
}{
	File:   "./caddy_schema.json",
	Indent: 2,
}

func init() {
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "json-schema",
		Func:  run,
		Usage: "[--output <file>] [--vscode] [--indent <int>]",
		Short: "Generate JSON schema for Caddy JSON api",
		Long: `
JSON schema generator for caddy JSON configuration.

If --output is set, the schema is generated to the specified file. By default it is
generated to caddy_schema.json in the current directory.

If --indent is set, the generated JSON files with be indented by n spaces where n is
the value of '--indent'.

If --vscode is set, schema and vscode config is generated into a '.vscode' directory
in the current working directory. This disregards '--output'.
Other ways of integrating JSON schema in VSCode can be found at
https://code.visualstudio.com/docs/languages/json#_mapping-in-the-user-settings
`,
		Flags: func() *flag.FlagSet {
			fs := flag.NewFlagSet("json-schema", flag.ExitOnError)
			fs.StringVar(&config.File, "output", config.File, "The file to write the generated schema")
			fs.BoolVar(&config.VsCode, "vscode", config.VsCode, "Generate VSCode configuration")
			fs.IntVar(&config.Indent, "indent", config.Indent, "Number of spaces to indent the generated JSON with")
			return fs
		}(),
	})
}

func run(fs caddycmd.Flags) (int, error) {
	if err := loadDoc(); err != nil {
		return caddy.ExitCodeFailedQuit, err
	}

	if err := generateSchema(); err != nil {
		return caddy.ExitCodeFailedQuit, err
	}

	var w schemaWriter
	{
		if config.VsCode {
			w = &vscodeWriter{}
		} else {
			w = basicWriter{}
		}
	}
	if err := writeToFile(w); err != nil {
		return caddy.ExitCodeFailedQuit, err
	}

	return 0, nil
}
