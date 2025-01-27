package parser

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"strconv"
	"encoding/csv"

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
//     set firewall ipv4 forward filter default-action 'accept'
//     set protocols static route 16.0.0.0/8 next-hop 10.250.0.1
//     set service ntp allow-client address '0.0.0.0/0'
//     set service ntp server 10.1.0.238
//     set service ssh port '22'
//     set system name-server '8.8.8.8'
//
// For my sample config, I never see TagNodes with quoted names, but
// LeafNodes *sometimes* are quoted and sometimes not.  Compare the
// `set service ntp server` and `set system name-server` lines -- they
// both contain an IP address here, but one is quoted and one isn't.
func ParseSetFormat(config string, configModel *configmodel.InterfaceDefinition) (*VyOSConfigAST, error) {
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

		csvReader := csv.NewReader(strings.NewReader(line))
		csvReader.Comma = ' '
		fields, err := csvReader.Read()
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

func parseSetLine(ast *VyOSConfigAST, fields []string, configModel *configmodel.InterfaceDefinition, lineno int) error {
	fmt.Printf("Got %v at line %d\n", fields, lineno)

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
	return strings.Join(results, "\n")+"\n", nil
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
