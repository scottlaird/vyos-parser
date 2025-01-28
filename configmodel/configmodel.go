package configmodel

// configmodel implements a parser for VyOS's XML configuration
// format.

// This file mostly contains code for handling VyOS's interface
// definition files, as found in 'vyos-1x/build/interface-definitions`
// after running `make.  Once it's parsed, we generally copy it into a
// VyOSConfigNode tree instead of using InterfaceDefinition directly,
// because that lets us get rid of the bulk of the special-case
// handling for Node/LeafNode/TagNode.


// InterfaceDefinition is the top-level definition of an config
// setting in VyOS's XML spec.  It should really be called
// `ConfigDefinition` or similar, but the XML tag that they use is
// `InterfaceDefinition` so I'm sticking with that.
type InterfaceDefinition struct {
        Nodes []*Node `xml:"node", json:"Node"`
}

func (id *InterfaceDefinition) VyOSConfig() *VyOSConfigNode {
        c := &VyOSConfigNode{
                Type: "root",
        }
        for _, n := range id.Nodes {
                c.Children = append(c.Children, n.VyOSConfigNode())
        }
        return c
}

// Node models the `<node>` tag in VyOS's XML config spec.  A node is
// basically a fixed string with no value in the middle of a config
// line.  So, with `interface ethernet eth0 address dhcp` the first
// `interface` is a Node, while `ethernet eth0` is a TagNode (with a
// parameter after the node name) and `address dhcp` is a LeafNode
// (with no children and an optional parameter).
type Node struct {
        Name       string          `xml:"name,attr" json:"name"`
        Owner      string          `xml:"owner,attr" json:"-"`
        Properties *NodeProperties `xml:"properties" json:"-"`
        Children   *NodeChildren   `xml:"children" json:"children"`
}

func (n *Node) VyOSConfigNode() *VyOSConfigNode {
        c := &VyOSConfigNode{
                Type: "node",
                Name: n.Name,
        }

        if n.Children != nil {
                c.Children = n.Children.VyOSConfigNode()
        }

        return c
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

func (nc *NodeChildren) VyOSConfigNode() []*VyOSConfigNode {
        children := []*VyOSConfigNode{}

        for _, ln := range nc.LeafNodes {
                children = append(children, ln.VyOSConfigNode())
        }
        for _, n := range nc.Nodes {
                children = append(children, n.VyOSConfigNode())
        }
        for _, tn := range nc.TagNodes {
                children = append(children, tn.VyOSConfigNode())
        }

        return children
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

func (ln *LeafNode) VyOSConfigNode() *VyOSConfigNode {
        c := &VyOSConfigNode{
                Type: "leafnode",
                Name: ln.Name,
        }

        value := true
        if ln.Properties != nil {
                if ln.Properties.Multi != nil {
                        c.Multi = true
                }
                if ln.Properties.Valueless != nil {
                        value = false
                }

        }
        c.HasValue = value

        return c
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

func (tn *TagNode) VyOSConfigNode() *VyOSConfigNode {
        c := &VyOSConfigNode{
                Type: "tagnode",
                Name: tn.Name,
        }

        // Always true for tagnodes?
        c.Multi = true
        c.HasValue = true

        if tn.Children != nil {
                c.Children = tn.Children.VyOSConfigNode()
        }

        return c
}
