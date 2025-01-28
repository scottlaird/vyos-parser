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
