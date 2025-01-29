package parser

import (
        "bufio"
        "fmt"
        "regexp"
        "strconv"
        "strings"

        "github.com/kballard/go-shellquote"
        "github.com/scottlaird/vyos-parser/configmodel"
)

var (
        unquotedRE = regexp.MustCompile("^[-_.:/+@a-zA-Z0-9]+$")
)

func quoteIfNeeded(value string) string {
        if unquotedRE.MatchString(value) {
                return value
        } else {
                s := strconv.Quote(value)
                // Unfortunately, VyOS uses single quotes in `show |
                // commands`, and I'd like to match it byte-for-byte
                // whenever possible.
                s = s[1 : len(s)-1] // remove quotes
                s = strings.ReplaceAll(s, "\\\"", "\"")
                s = strings.ReplaceAll(s, "'", "\\'")
                s = "'" + s + "'"
                return s
        }
}

// ParseSetFormat takes a VyOS text configuration in
// `set` format and returns a VyOSConfigAST and/or an error.
//
// Note that VyOS's config outputter (`show | commands`) isn't very
// consistent about quoting.  Examples, from VyOS 1.5 202501xxx:
//
//      set firewall ipv4 forward filter default-action 'accept'
//      set protocols static route 16.0.0.0/8 next-hop 10.250.0.1
//      set service ntp allow-client address '0.0.0.0/0'
//      set service ntp server 10.1.0.238
//      set service ssh port '22'
//      set system name-server '8.8.8.8'
//
// For my sample config, I never see TagNodes with quoted names, but
// LeafNodes *sometimes* are quoted and sometimes not.  Compare the
// `set service ntp server` and `set system name-server` lines -- they
// both contain an IP address here, but one is quoted and one isn't.
func ParseSetFormat(config string, configModel *configmodel.VyOSConfigNode) (*VyOSConfigAST, error) {
        ast := &VyOSConfigAST{}
        child := &Node{
                Type: "root",
        }
        ast.Child = child
        scanner := bufio.NewScanner(strings.NewReader(config))

        if err := scanner.Err(); err != nil {
                return ast, fmt.Errorf("Error occurred while scanning: %v", err)
        }

        lineno := 0
        for scanner.Scan() {
                line := strings.TrimSpace(scanner.Text())
                lineno++

                fields, err := shellquote.Split(line)
                if err != nil {
                        return nil, fmt.Errorf("Failed to split %q by words at line %d: %v", line, lineno, err)
                }

                err = parseSetLine(ast, fields, configModel, lineno)
                if err != nil {
                        return nil, err // lineno should be in err already
                }
        }

        return ast, nil
}

// Unquote removes quotes from a string; for double quotes
// strconv.Unquote would work, but that doesn't work for single
// quotes.
func unquote(s string) string {
        // For double quotes, use strconv.Unquote
        if s[0] == '"' {
                u, err := strconv.Unquote(s)
                if err == nil {
                        return u
                }
        }

        // For single quotes, just strip the first and last character.
        // This won't correctly handle escaped quotes, etc, but it's
        // probably close enough for now.
        if s[0] == '\'' {
                return s[1 : len(s)-1]
        }

        return s
}

func parseSetLine(ast *VyOSConfigAST, fields []string, configModel *configmodel.VyOSConfigNode, lineno int) error {
        // Allow blank lines and shell or C++ comments
        if len(fields) == 0 || len(fields[0]) == 0 || fields[0][0] == '#' || fields[0][0:2] == "//" {
                return nil
        }

        if fields[0] != "set" {
                return fmt.Errorf("First word is not 'set' at line %d", lineno)
        }

        i := 1
        length := len(fields)
        configNode := configModel
        node := ast.Child
        errorPath := []string{}

        for {
                var value *string
                if i >= length {
                        break
                }
                field := unquote(fields[i])
                newConfigNode := configNode.FindNodeByName(field)
                if newConfigNode == nil {
                        options := []string{}
                        for _, n := range configNode.Children {
                                options = append(options, n.Name)
                        }
                        return fmt.Errorf("Unexpected word %q at line %d (options for %q are %v)", field, lineno, strings.Join(errorPath, " > "), options)
                }

                configNode = newConfigNode

                if configNode.HasValue {
                        i++
                        v := unquote(fields[i])
                        value = &v
                        errorPath = append(errorPath, field+" "+v)
                } else {
                        errorPath = append(errorPath, field)
                }

                newnode := node.addNode(configNode, value)
                newnode.Value = value
                node = newnode

                i++
        }

        return nil
}

// WriteSetFormat returns a string that contains the `set` format of
// the specified config AST.  That is, it writes out a bunch of `set
// ...` command lines that can be copied into VyOS to tell it to
// configure itself a specific way.
func WriteSetFormat(ast *VyOSConfigAST) (string, error) {
        results, err := writeSetPartial(ast.Child, "set")
        if err != nil {
                return "", err
        }
        return strings.Join(results, "\n") + "\n", nil
}

// writeSetPartial recursively turns AST nodes into `set ...` strings.
func writeSetPartial(node *Node, context string) ([]string, error) {
        results := []string{}
        if node.ContextNode != nil {
                context = context + " " + node.ContextNode.Name
        }

        if node.Value != nil {
                if node.Type == "leafnode" {
                        // VyOS always single-quotes LeafNode values
                        context = context + " '" + *node.Value + "'"
                } else {
                        // VyOS presumably still quotes TagNode values
                        // if they have spaces and/or shell
                        // metacharacters.
                        context = context + " " + quoteIfNeeded(*node.Value)
                }
        }

        for _, child := range node.Children {
                childresults, err := writeSetPartial(child, context)
                if err != nil {
                        return nil, err
                }
                results = append(results, childresults...)
        }
        if len(node.Children) == 0 {
                results = append(results, context)
        }
        return results, nil
}
