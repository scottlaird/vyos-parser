package parser

import (
	"fmt"
	"os"
	"testing"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/scottlaird/vyos-parser/syntax"
	"github.com/scottlaird/vyos-parser/configmodel"
)

func getConfigModel(t *testing.T) *configmodel.VyOSConfigNode {
	configModel, err := syntax.GetDefaultConfigModel()
	if err != nil {
		t.Fatalf("Failed to open configmodel JSON file: %v", err)
	}
	return configModel
}

func TestParseSetComment(t *testing.T) {
	configModel := getConfigModel(t)

	_, err := ParseSetFormat(`# not a set command`, configModel)
	if err != nil {
		t.Errorf("Failed to parse shell comment: %v", err)
	}

	_, err = ParseSetFormat(`// still not a set command`, configModel)
	if err != nil {
		t.Errorf("Failed to parse C++-style comment: %v", err)
	}

	_, err = ParseSetFormat(`/* not a valid comment */`, configModel)
	if err == nil {
		t.Errorf("Failed to error on C-style comment: %v", err)
	}
}

func TestParseSetRoundTrip(t *testing.T) {
	filename := "testdata/config.set.1"
	configModel := getConfigModel(t)

	b, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to open testdata file: %v", err)
	}

	originalSetConfig := string(b)

	ast, err := ParseSetFormat(originalSetConfig, configModel)
	if err != nil {
		t.Fatalf("Failed to parse %s: %v", filename, err)
	}

	newSetConfig, err := WriteSetFormat(ast)
	if err != nil {
		t.Fatalf("Failed calling writeSetFormat: %v", err)
	}

	if newSetConfig != originalSetConfig {
		edits := myers.ComputeEdits("foo", originalSetConfig, newSetConfig)
		fmt.Println(gotextdiff.ToUnified("config.set.1", "output", originalSetConfig, edits))
		t.Errorf("Generated set-format file does not match config.set.1")
	}
}
