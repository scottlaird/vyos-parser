package configmodel

import (
	"encoding/json"
	"io/fs"
	"os"
)

type VyOSConfigNode struct {
	Type     string            `json:"type,omitempty"`
	Name     string            `json:"name"`
	Children []*VyOSConfigNode `json:"children,omitempty"`
	Multi    *bool             `json:"multi,omitempty"`
	HasValue *bool             `json:"has_value,omitempty"`
}

func (vcn *VyOSConfigNode) FindNodeByName(name string) *VyOSConfigNode {
	for _, n := range vcn.Children {
		if n.Name == name {
			return n
		}
	}
	return nil
}

// Write ConfigModel to JSON file
func (vcn *VyOSConfigNode) WriteJSONFile(filename string, umask fs.FileMode) error {
	b, err := json.MarshalIndent(vcn, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, b, umask)
}

// Merge the children of 2 nodes together
func (vcn *VyOSConfigNode) Merge(b *VyOSConfigNode) {
OUTER:
	for _, node2 := range b.Children {
		for _, node1 := range vcn.Children {
			if node1.Name == node2.Name {
				node1.Merge(node2)
				continue OUTER
			}
		}
		vcn.Children = append(vcn.Children, node2)
	}
}

// Read ConfigModel from a JSON file
func LoadJSONFile(filename string) (*VyOSConfigNode, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	id := &VyOSConfigNode{}
	err = json.Unmarshal(b, &id)
	return id, err
}
