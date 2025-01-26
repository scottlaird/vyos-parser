package configmodel

import (
	"encoding/json"
	"os"
	"io/fs"
)

type VyOSConfigNode struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name"`
	Children []*VyOSConfigNode `json:"children,omitempty"`
	Multi *bool `json:"multi,omitempty"`
	HasValue *bool `json:"has_value,omitempty"`
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
