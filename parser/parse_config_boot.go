package parser

import (
        "bufio"
        "fmt"
        "strings"
        "strconv"

        "github.com/scottlaird/vyos-parser/configmodel"
)


// ParseConfigBootFormat takes a VyOS text configuration in
// `config.boot` format and returns a VyOSConfigAST and/or an error.
func ParseConfigBootFormat(config string, configModel *configmodel.VyOSConfigNode) (*VyOSConfigAST, error) {
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
        err := parseConfigBootFormat(child, scanner, configModel, &lineno)
        return ast, err
}

// parseConfigBootFormat parses everything underneath a higher-level
// node in the config.  It reads from the `config` scanner and updates
// `nodeContext` as needed, returning an error if it's unable to parse.
func parseConfigBootFormat(nodeContext *Node, scanner *bufio.Scanner, configModel *configmodel.VyOSConfigNode, lineno *int) error {
        for scanner.Scan() {
                line := scanner.Text()
                (*lineno)++
                
                line = strings.TrimSpace(line)

                if len(line)==0 || line[0]=='/' {
                        continue
                }
                
                if line[0] == '}' {
                        return nil
                }

                name := strings.SplitN(line, " ", 2)[0]

                configNode := configModel.FindNodeByName(name)
                if configNode == nil {
                        return fmt.Errorf("Couldn't match config token %q at line number %d", name, *lineno)
                }

                // This can't use `addNode` because that tries to
                // merge children with the same name, which breaks
                // with TagNodes or LeafNodes with multi=true
                astNode := newASTNode(configNode)
                nodeContext.Children = append(nodeContext.Children, astNode)
                
                remainingLine := line[len(name):]
                if len(remainingLine)>0 {
                        if remainingLine[len(remainingLine)-1] == '{' {
                                remainingLine = remainingLine[:len(remainingLine)-1] // strip "{" and whitespace from the end.
                        }
                        remainingLine = strings.TrimSpace(remainingLine)

                        if len(remainingLine)>0 && remainingLine[0]=='"' {
                                value, err := strconv.Unquote(remainingLine)
                                if err != nil {
                                        fmt.Printf("Unquote error: %v\n", err)
                                        return err
                                }
                                astNode.Value = &value
                        } else {
                                if len(remainingLine)>0 {
                                        // Not completely happy about this
                                        astNode.Value = &remainingLine
                                        //fmt.Printf("Found extra value of %q in %q at line %d\n", remainingLine, line, *lineno)
                                }
                        }
                }

                
                if line[len(line)-1] == '{' {
                        parseConfigBootFormat(astNode, scanner, configNode, lineno)
                }
        }

        // Ran out of text?
        return nil
}

func WriteConfigBootFormat(ast *VyOSConfigAST) (string, error) {
        results, err := writeConfigBootPartial(ast.Child, 0)
        if err != nil {
                return "", err
        }
        return strings.Join(results, "\n")+"\n", nil
}

// spaces returns a string with a specific number of space characters,
// used for indenting.
func spaces(indent int) string {
        return fmt.Sprintf("%*s", indent, "")
}

func writeConfigBootPartial(node *Node, indent int) ([]string, error) {
        results := []string{}
        line := ""
        newIndent := indent + 4

        if node.ContextNode != nil {
                line = node.ContextNode.Name
                
                if node.Value != nil && *node.Value != "" {
                        value := *node.Value
                        // Leafnodes get values in double quotes
                        if node.Type == "leafnode" {
                                value = "\"" + value + "\""
                        }
                        line = line + " " + value
                }
        } else {
                newIndent = indent
        }
        

        if node.Type == "leafnode" {
                results = append(results, spaces(indent) + line)
        } else {
                if line != "" {
                        results = append(results, spaces(indent) + line + " {" )
                }
                for _, n := range node.Children {
                        res, err := writeConfigBootPartial(n, newIndent)
                        if err != nil {
                                return []string{}, err
                        }
                        
                        results = append(results, res...)
                }
                if line != "" {
                        results = append(results, spaces(indent) + "}")
                }
        }
        return results, nil
}
