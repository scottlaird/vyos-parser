This is a Go library for parsing config files for
[VyOS](https://vyos.io) router configs.

VyOS's configs are similar (but not exactly the same) to Juniper's
Junos configs; they're hierarchial and block-structured.  Like Junos,
there are two main forms for configs:

- The version returned from `show configuration` (or just `show` in config mode)
- A version that consists of `set` commands that can re-create any
  given configuration.  In VyOS, you get generate this form via `show
  | commands` in config mode.

The second mode has the advantage of being trivial to cut and paste
into a router, but it's generally a bit harder to read and much harder
to edit.

VyOS also adds a third config format, although it's mostly hidden
behind the scenes; it stores its boot config in `/config/config.boot`
in *almost* the same format as "show" outputs, but with different
quoting rules.

This library supports reading and writing all three formats.  Here's a
quick example that will read a "show" format config from a file and
print the "set" equivalent to stdout:

```go
  package main
  
  import (
      "fmt"
      "os"
      
      "github.com/scottlaird/vyos-parser/syntax"
      "github.com/scottlaird/vyos-parser/parser"
  )
  
  func main() {
      // Error checking omitted
      configModel, _ := syntax.GetDefaultConfigModel()
      config, _ := os.ReadFile("my-show-config.txt")
      ast, _ := parser.ParseShowFormat(string(config), configModel)
      setConfig, _ := parser.WriteSetFormat(ast)
      fmt.Println(setConfig)
  }
```

At least for trivial configs, this library should be able to produce
byte-for-byte identical output that matches what VyOS
1.5-rolling-202501060800 produces, with the exception of version
migration comments at the end of `config.boot`.

## Syntax

This library uses VyOS's own syntax definitions in parsing.  They're
extracted from `vyos-1x/interface-definitions`, parsed, and dropped
into `syntax/vyos*.json.gz`.  To produce a newer syntax definition,
run `git pull` in the `vyos-1x` subdirectory, then run `make` in the
`vyos-parser` root directory.  This will produce a new definition file
in `synatax/`, named using the date of the most recent commit to
`vyos-1x`.  To make this the default, edit `syntax/default.version`.
