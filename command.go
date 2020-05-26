package jsonschema

import (
	"flag"
	stdlog "log"
	"os"

	"github.com/caddyserver/caddy/v2"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"
)

const commandName = "json-schema"

var (
	config = struct {
		File         string
		VsCode       bool
		Indent       int
		DiscardCache bool
	}{
		File:   "./caddy_schema.json",
		Indent: 2,
	}

	log = stdlog.New(os.Stderr, commandName+" ", 0)
)

func init() {
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  commandName,
		Func:  run,
		Usage: "[--output <file>] [--indent <int>] [--vscode] [--no-cache]",
		Short: "Generate JSON schema for Caddy JSON api",
		Long: `
JSON schema generator for caddy JSON configuration.

If --output is set, the schema is generated to the specified file. By default it is
generated to caddy_schema.json in the current directory.

If --indent is set, the generated JSON files with be indented by n spaces where n is
the value of '--indent'.

If --no-cache is set, local documentation cache (if present) will be discard and the
latest API docs will be retrieved from caddyserver.com.

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
			fs.BoolVar(&config.DiscardCache, "no-cache", config.DiscardCache, "Discard local cache and fetch latest API docs")
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
