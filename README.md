# caddy-json-schema

JSON schema generator for Caddy v2.

The generated schema can be integrated with editors for intellisense and better experience with configuration and plugin development.

![Demonstration](https://github.com/abiosoft/caddy-json-schema/blob/master/gif/schema.gif)

## Installation

The generated schema is for the caddy binary. i.e. all modules in the binary will
be include in the schema.

```sh
xcaddy build v2.0.0 \
    --with github.com/abiosoft/caddy-json-schema \
    # any other module you want to include in the generated schema
```

## Usage

Run `caddy help json-schema` to view help.

```
usage:
  caddy json-schema [--output <file>] [--indent <int>] [--vscode] [--no-cache]

flags:
  -indent int
        Number of spaces to indent the generated JSON with (default 2)
  -no-cache
        Discard local cache and fetch latest API docs
  -output string
        The file to write the generated schema (default "./caddy_schema.json")
  -vscode
        Generate VSCode configuration
```

## Editors

### Visual Studio Code

`caddy json-schema --vscode` generates Visual Studio Code configuration in the current directory.

Open the directory in Visual Studio Code and it should just work. 
Ensure the config filename is of the format `*caddy*.json`.

### Vim/NeoVim

There are multiple Vim/NeoVim plugins with language server and JSON schema support.

Below is a config for [coc-json](https://github.com/neoclide/coc-json). The `url` is a relative path to the config file being edited.

```json
{
    "json.schemas": [
        {
            "fileMatch": [
                "*caddy*.json"
            ],
            "url": "./caddy_schema.json"
        }
    ]

}
```

## Features

| Modules | Intellisense | Documentation |
|---------|--------------|---------------|
| Standard| Supported | Supported |
| Third Party| Supported | Planned |

## License

Apache 2
