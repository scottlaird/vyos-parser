package parser

import (
        "bufio"
        "fmt"
        "strings"
        "strconv"
        "regexp"

        "github.com/scottlaird/vyos-parser/configmodel"
)

var (
        showQuoteRE = regexp.MustCompile("^[-_.:/+@$a-zA-Z0-9]+$")
)

func doubleQuoteIfNeeded(value string) string {
        if showQuoteRE.MatchString(value) {
                return value
        } else {
                return strconv.Quote(value)
        }
}

// ParseShowFormat takes a VyOS text configuration in the format
// returned by 'show' from config mode and returns a VyOSConfigAST
// and/or an error.
func ParseShowFormat(config string, configModel *configmodel.VyOSConfigNode) (*VyOSConfigAST, error) {
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
        err := parseShowFormat(child, scanner, configModel, &lineno)
        return ast, err
}

// parseShowFormat parses everything underneath a higher-level
// node in the config.  It reads from the `config` scanner and updates
// `nodeContext` as needed, returning an error if it's unable to parse.
func parseShowFormat(nodeContext *Node, scanner *bufio.Scanner, configModel *configmodel.VyOSConfigNode, lineno *int) error {
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

                // Figure out if there's a value associated with this node
                remainingLine := line[len(name):]
                value := ""
                if len(remainingLine)>0 {
                        if remainingLine[len(remainingLine)-1] == '{' {
                                remainingLine = remainingLine[:len(remainingLine)-1] // strip "{" and whitespace from the end.
                        }
                        remainingLine = strings.TrimSpace(remainingLine)

                        // Deal with quoting
                        if len(remainingLine)>0 && remainingLine[0]=='"' {
                                val, err := strconv.Unquote(remainingLine)
                                if err != nil {
                                        fmt.Printf("Unquote error: %v\n", err)
                                        return err
                                }
                                value = val
                        } else {
                                if len(remainingLine)>0 {
                                        // Not completely happy about this
                                        value = remainingLine
                                        //fmt.Printf("Found extra value of %q in %q at line %d\n", remainingLine, line, *lineno)
                                }
                        }
                }

                var astNode *Node

                // Check if this is a duplicate of an existing node
                if configNode.Multi {
                        // This node allows multiple values
                        for _, n := range nodeContext.Children {
                                if n.ContextNode.Name==name && *n.Value == value {
                                        astNode = n
                                }
                        }
                } else {
                        // This node does *not* allow multiple values
                        for _, n := range nodeContext.Children {
                                if n.ContextNode.Name==name {
                                        astNode = n
                                }
                        }
                }

                // Not a duplicate, so create a new node
                if astNode == nil {
                        astNode = newASTNode(configNode)
                        nodeContext.Children = append(nodeContext.Children, astNode)
                }

                if value != "" {
                        astNode.Value = &value
                }
                
                if line[len(line)-1] == '{' {
                        parseShowFormat(astNode, scanner, configNode, lineno)
                }
        }

        // Ran out of text?
        return nil
}

func WriteShowFormat(ast *VyOSConfigAST) (string, error) {
        results, err := writeShowPartial(ast.Child, 1)
        if err != nil {
                return "", err
        }
        return strings.Join(results, "\n")+"\n", nil
}

func writeShowPartial(node *Node, indent int) ([]string, error) {
        results := []string{}
        line := ""
        newIndent := indent + 4

        if node.ContextNode != nil {
                line = node.ContextNode.Name
                
                if node.Value != nil && *node.Value != "" {
                        value := *node.Value
                        line = line + " " + doubleQuoteIfNeeded(value)
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
                        res, err := writeShowPartial(n, newIndent)
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
