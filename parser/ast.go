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
