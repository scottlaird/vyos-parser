package parser

import (
	"testing"
	"os"
	"fmt"

	"github.com/scottlaird/vyos-parser/syntax"
    "github.com/hexops/gotextdiff"
    "github.com/hexops/gotextdiff/myers"
)

func TestParseConfigBootFormat(t *testing.T) {
	configModel, err := syntax.GetDefaultConfigModel()
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

	setFile, err := os.ReadFile("testdata/config.set.1")
	if err != nil {
		t.Fatalf("Failed to open config.set.1: %v", err)
	}

	if set != string(setFile) {
		edits := myers.ComputeEdits("foo", string(setFile), set)
		fmt.Println(gotextdiff.ToUnified("config.set.1", "output", string(setFile), edits))
		t.Errorf("Generated set-format file does not match config.set.1")

	}

}
