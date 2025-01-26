package configmodel

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
)

const (
	GENERICTYPE_ROOT = iota
	GENERICTYPE_NODE
	GENERICTYPE_LEAFNODE
	GENERICTYPE_TAGNODE
)

// spaces returns a string with a specific number of space characters,
// used for indenting.
func spaces(indent int) string {
	return fmt.Sprintf("%*s", indent, "")
}

// There are 3 node types in here (Node, LeafNode, TagNode), and I
// don't want to have to keep re-writing common functions.  This
// interface covers all 3 types.  Note that it's not `nodeType`, it's
// `nodeType[T any]`, as this allows us to say that the `Merge()`
// method takes a parameter of the same type.
type nodeType[T any] interface {
	GetName() string
	Merge(T)
	Print(int)
}

// mergeCollections merges two arrays of nodes and returns the result.
// Any nodes that are in the second parameter (`b`) but not in the
// first (`a`) are appended to `a`.  If both collections have a node
// of the same name, then the nodes will be merged recursively.
//
// This allows us to merge VyOS's partial syntax XML files into a
// single grammar.  For instance `interface ethernet XXX` and
// `interface bridge XXX` are in different files.  So, when we merge
// the top-level nodes, we find that they both have `interface`, and
// then merge that recursively.  Under that, they both have `TagNode`s
// that differ, so they're appended together, resulting in a single
// syntax that allows for both types of interfaces.
//
// By doing this process across all ~120 XML files that define VyOS's
// syntax, we're able to assemble a complete copy of their config
// language pretty easily.
func mergeNodes[T nodeType[T]](a, b []T) []T {
OUTER:
	for _, node2 := range b {
		for _, node1 := range a {
			if node1.GetName() == node2.GetName() {
				node1.Merge(node2)
				continue OUTER
			}
		}
		a = append(a, node2)
	}

	return a
}

// InterfaceDefinition is the top-level definition of an config
// setting in VyOS's XML spec.  It should really be called
// `ConfigDefinition` or similar, but the XML tag that they use is
// `InterfaceDefinition` so I'm sticking with that.
type InterfaceDefinition struct {
	Nodes []*Node `xml:"node", json:"Node"`
}

// Recursively fix field definitions
func (id *InterfaceDefinition) Fixup() {
	for _, n := range id.Nodes {
		n.Fixup()
	}
}

// Print an InterfaceDefinition
func (id *InterfaceDefinition) Print(indent int) {
	fmt.Printf("%sInterfaceDefinition\n", spaces(indent))
	for _, n := range id.Nodes {
		n.Print(indent + 2)
	}
}

// Merge two InterfaceDefinitions
func (id *InterfaceDefinition) Merge(id2 *InterfaceDefinition) {
	id.Nodes = mergeNodes(id.Nodes, id2.Nodes)
}

// Write ConfigModel to JSON file
func (id *InterfaceDefinition) WriteJSONFile(filename string, umask fs.FileMode) error {
	b, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, b, umask)
}

func (id *InterfaceDefinition) Generic() *GenericNode {
	return &GenericNode{
		NodeType: GENERICTYPE_ROOT,
		id: id,
	}
}

func (id *InterfaceDefinition) FindNodeByName(name string) *GenericNode {
	for _, n := range id.Nodes {
		if n.Name == name {
			return n.Generic()
		}
	}
	return nil
}


// Read ConfigModel from a JSON file
func LoadJSONFile(filename string) (*InterfaceDefinition, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	id := &InterfaceDefinition{}
	err = json.Unmarshal(b, &id)
	return id, err
}

// A GenericNode can be a Node, a LeafNode, or a TagNode
type GenericNode struct {
	NodeType int
	n *Node
	ln *LeafNode
	tn *TagNode
	id *InterfaceDefinition // root
}

// Find a specific node by name
func (gn *GenericNode) FindNodeByName(name string) *GenericNode {
	switch gn.NodeType {
	case GENERICTYPE_ROOT:
		return gn.id.FindNodeByName(name)
	case GENERICTYPE_NODE:
		return gn.n.FindNodeByName(name)
	case GENERICTYPE_LEAFNODE:
		return gn.ln.FindNodeByName(name)
	case GENERICTYPE_TAGNODE:
		return gn.tn.FindNodeByName(name)
	default:
		return nil
	}
}

// GetName
func (gn *GenericNode) GetName() string {
	switch gn.NodeType {
	case GENERICTYPE_ROOT:
		return ""
	case GENERICTYPE_NODE:
		return gn.n.GetName()
	case GENERICTYPE_LEAFNODE:
		return gn.ln.GetName()
	case GENERICTYPE_TAGNODE:
		return gn.tn.GetName()
	default:
		return "UNKNOWN"
	}
}



// Node models the `<node>` tag in VyOS's XML config spec.  A node is
// basically a fixed string with no value in the middle of a config
// line.  So, with `interface ethernet eth0 address dhcp` the first
// `interface` is a Node, while `ethernet eth0` is a TagNode and
// `address dhcp` is a LeafNode.
type Node struct {
	Name       string          `xml:"name,attr" json:"name"`
	Owner      string          `xml:"owner,attr" json:"-"`
	Properties *NodeProperties `xml:"properties" json:"-"`
	Children   *NodeChildren   `xml:"children" json:"children"`
}

func (n *Node) Fixup() {
	// Nothing to do for Node yet.

	if n.Children != nil {
		n.Children.Fixup()
	}
}

func (n *Node) GetName() string { return n.Name }

func (n *Node) Print(indent int) {
	fmt.Printf("%sNode(%q)\n", spaces(indent), n.Name)
	// Not printing properties right now
	n.Children.Print(indent + 2)
}

func (n *Node) Merge(n2 *Node) {
	n.Children.Merge(n2.Children)
}

func (n *Node) Generic() *GenericNode {
	return &GenericNode{
		NodeType: GENERICTYPE_NODE,
		n: n,
	}
}

func (n *Node) FindNodeByName(name string) *GenericNode {
	return n.Children.FindNodeByName(name)
}

// NodeProperties model the `<properties>` tag in VyOS's XML config
// spec.  This includes help text, validation options, and a number of
// other things that we don't actually care about at the moment.  It
// also contains two booleans, `Multi` and `Valueless` that tell us
// when a Node (or generally a LeafNode) take either multiple values
// or no values at all.
type NodeProperties struct {
	DefaultValue   string                    `xml:"defaultValue" json:"-"`
	Help           []*PropertyHelp           `xml:"help" json:"-"`
	CompletionHelp []*PropertyCompletionHelp `xml:"completionHelp" json:"-"`
	ValueHelp      []*PropertyValueHelp      `xml:"valueHelp" json:"-"`
	Constraint     []*PropertyConstraint     `xml:"constraint" json:"-"`
	//ConstraintErrorMessage string `xml:"constraintErrorMessage,chardata"`
	Multi     *bool `xml:"multi" json:"multi,omitempty"`
	Valueless *bool `xml:"valueless" json:"valueless,omitempty"`
}

func (np *NodeProperties) valueType() string {
	if np == nil {
		return "NIL"
	}

	if np.Multi != nil {
		return "VALUE..."
	}
	if np.Valueless != nil {
		return ""
	}
	return "VALUE"
}

type PropertyHelp struct {
	Text string `xml:",chardata"`
}

type PropertyCompletionHelp struct {
	InnerXML string `xml:",innerxml"` // Just collect for now.
}

type PropertyValueHelp struct {
	InnerXML string `xml:",innerxml"` // Just collect for now.
}

type PropertyConstraint struct {
	InnerXML string `xml:",innerxml"` // Just collect for now.
}

// NodeChildren models the `<children>` tag inside of the various node
// types in VyOS's XML config spec.  There are three types of nodes,
// each contained in their own list.
type NodeChildren struct {
	LeafNodes []*LeafNode `xml:"leafNode" json:"LeafNodes,omitempty"`
	Nodes     []*Node     `xml:"node" json:"Nodes,omitempty"`
	TagNodes  []*TagNode  `xml:"tagNode" json:"TagNodes,omitempty"`
}

func (nc *NodeChildren) Fixup() {
	for _, ln := range nc.LeafNodes {
		ln.Fixup()
	}
	for _, n := range nc.Nodes {
		n.Fixup()
	}
	for _, tn := range nc.TagNodes {
		tn.Fixup()
	}
}

// Print prints the children of a node.
func (nc *NodeChildren) Print(indent int) {
	for _, ln := range nc.LeafNodes {
		ln.Print(indent)
	}
	for _, n := range nc.Nodes {
		n.Print(indent)
	}
	for _, tn := range nc.TagNodes {
		tn.Print(indent)
	}
}

// Merge merges two NodeChildren structs.
func (nc *NodeChildren) Merge(nc2 *NodeChildren) {
	nc.LeafNodes = mergeNodes(nc.LeafNodes, nc2.LeafNodes)
	nc.Nodes = mergeNodes(nc.Nodes, nc2.Nodes)
	nc.TagNodes = mergeNodes(nc.TagNodes, nc2.TagNodes)
}


func (nc *NodeChildren) FindNodeByName(name string) *GenericNode {
	for _, n := range nc.Nodes {
		if n.Name == name {
			return n.Generic()
		}
	}
	for _, ln := range nc.LeafNodes {
		if ln.Name == name {
			return ln.Generic()
		}
	}
	for _, tn := range nc.TagNodes {
		if tn.Name == name {
			return tn.Generic()
		}
	}
	return nil
}

// LeafNode models the `<leafNode>` tag in VyOS's XML config spec.  A
// leafNode is a terminal node with no children and optionally a
// single parameter or list of parameters.
type LeafNode struct {
	Name       string          `xml:"name,attr" json:"name"`
	Owner      string          `xml:"owner,attr" json:"-"`
	Properties *NodeProperties `xml:"properties" json:"-"`
	ValueType  string          `json:"valuetype"`
}

func (ln *LeafNode) Fixup() {
	if ln.Properties == nil {
		ln.ValueType = "SINGLE"
	} else if ln.Properties.Valueless != nil {
		ln.ValueType = "NONE"
	} else if ln.Properties.Multi != nil {
		ln.ValueType = "MULTI"
	} else {
		ln.ValueType = "SINGLE"
	}
}

func (ln *LeafNode) GetName() string { return ln.Name }

func (ln *LeafNode) Print(indent int) {
	fmt.Printf("%sLeafNode(%q) %s\n", spaces(indent), ln.Name, ln.Properties.valueType())
}

func (ln *LeafNode) Merge(n2 *LeafNode) {
	return // leafnodes don't have children
}

func (ln *LeafNode) Generic() *GenericNode {
	return &GenericNode{
		NodeType: GENERICTYPE_LEAFNODE,
		ln: ln,
	}
}

func (ln *LeafNode) FindNodeByName(name string) *GenericNode {
	return nil
}

// TagNode models the `<tagNode>` tag in VyOS's XML config spec.  A
// tagNode is a node in the middle of the config space that takes a
// value; in `interface ethernet eth0 address dhcp`, the `ethernet
// eth0` is a tagNode, with a value of `eth0`.
type TagNode struct {
	Name       string          `xml:"name,attr" json:"name"`
	Owner      string          `xml:"owner,attr" json:"-"`
	Properties *NodeProperties `xml:"properties" json:"-"`
	Children   *NodeChildren   `xml:"children" json:"children"`
}

func (tn *TagNode) Fixup() {
	if tn.Children != nil {
		tn.Children.Fixup()
	}
}

func (tn *TagNode) GetName() string { return tn.Name }

func (tn *TagNode) Print(indent int) {
	fmt.Printf("%sTagNode(%q) %s\n", spaces(indent), tn.Name, tn.Properties.valueType())
	// Not printing properties right now
	tn.Children.Print(indent + 2)
}

func (tn *TagNode) Merge(tn2 *TagNode) {
	tn.Children.Merge(tn2.Children)
}

func (tn *TagNode) Generic() *GenericNode {
	return &GenericNode{
		NodeType: GENERICTYPE_TAGNODE,
		tn: tn,
	}
}

func (tn *TagNode) FindNodeByName(name string) *GenericNode {
	return tn.Children.FindNodeByName(name)
}
