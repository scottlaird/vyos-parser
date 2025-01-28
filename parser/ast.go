package parser

import (
        "slices"
        "cmp"
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

func (vca *VyOSConfigAST) Sort() {
        vca.Child.Sort()
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

func (n *Node) Sort() {
        slices.SortFunc(n.Children, func(a, b *Node) int {
                names := cmp.Compare(a.ContextNode.Name, b.ContextNode.Name)
                if names != 0 {
                        return names
                }
                if a.Value != nil && b.Value != nil {
                        return cmp.Compare(*a.Value, *b.Value)
                }
                return 0
        })

        for _, child := range n.Children {
                child.Sort()
        }
}

func newASTNode(contextNode *configmodel.VyOSConfigNode) *Node {
        return &Node{
                Type: contextNode.Type,
                ContextNode: contextNode,
        }
}

func (n *Node) addNode(contextNode *configmodel.VyOSConfigNode, value *string) *Node {
        if contextNode.Multi {
                // This node allows for multiple entries, so we need
                // to make sure that we're looking at the *correct*
                // child node
                for _, child := range n.Children {
                        if child.ContextNode != nil && child.ContextNode.Name == contextNode.Name {
                                if *child.Value == *value {
                                        return child
                                }
                        }
                }
        } else {
                // This is a single-entry child, so we want to
                // overwrite the previous value.
                for _, child := range n.Children {
                        if child.ContextNode != nil && child.ContextNode.Name == contextNode.Name {
                                child.Value = value
                                return child
                        }
                }
        }
        child := newASTNode(contextNode)
        child.Value = value
        n.Children = append(n.Children, child)
        return child
}
