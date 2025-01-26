package parser

import (
	"testing"
	"os"
	"fmt"

	"github.com/scottlaird/vyos-parser/configmodel"
)

func TestParseConfigBootFormat(t *testing.T) {
	configModel, err := configmodel.LoadJSONFile("../vyos.json")
	if err != nil {
		t.Fatalf("Failed to open configmodel JSON file: %v", err)
	}
	
	b, err := os.ReadFile("testdata/config.boot.1")
	if err != nil {
		t.Fatalf("Failed to open testdata file: %v", err)
	}
	ast, err := ParseConfigBootFormat(string(b), configModel)
	if err != nil {
		t.Fatalf("Failed to parse config.boot.1: %v", err)
	}

	treesize := ast.TreeSize()
	want := 162
	if treesize != want {
		t.Errorf("Got treesize=%d, want %d", treesize, want)
	}

	set, err := WriteSetFormat(ast)
	if err != nil {
		t.Fatalf("Failed calling writeSetFormat: %v", err)
	}

	fmt.Printf("Set is:\n%s\n", set)
}
