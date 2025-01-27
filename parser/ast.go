package parser

import (
		"github.com/scottlaird/vyos-parser/configmodel"
)

const (
	NodeTypeRoot = iota
	NodeTypeNode
	NodeTypeLeaf
	NodeTypeTag
)

type VyOSConfigAST struct {
	Child *Node
}

func (vca *VyOSConfigAST) TreeSize() int {
	return vca.Child.TreeSize()
}

type Node struct {
	ContextNode *configmodel.VyOSConfigNode
	Type string
	Value *string
	Children []*Node
}

func (n *Node) TreeSize() int {
	size := 1

	for _, child := range n.Children {
		size += child.TreeSize()
	}
	return size
}

func newASTNode(contextNode *configmodel.VyOSConfigNode) *Node {
	return &Node{
		Type: contextNode.Type,
		ContextNode: contextNode,
	}
}

func (n *Node) addNode(contextNode *configmodel.VyOSConfigNode, value *string) *Node {
	for _, child := range n.Children {
		if child.ContextNode != nil && child.ContextNode.Name == contextNode.Name {
			if child.Value == value {
				return child
			}
			// Should probably do something with 'Multi'
			// here, but we're not actually trying to
			// validate configs, just transform them.
		}
	}
	child := newASTNode(contextNode)
	child.Value = value
	n.Children = append(n.Children, child)
	return child
}
