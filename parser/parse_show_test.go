package parser

import (
        "fmt"
        "os"
        "strings"
        "testing"

        "github.com/hexops/gotextdiff"
        "github.com/hexops/gotextdiff/myers"
)

func TestParseShowRoundTrip(t *testing.T) {
        configModel := getConfigModel(t)
        filename := "testdata/config.show.1"

        b, err := os.ReadFile(filename)
        if err != nil {
                t.Fatalf("Failed to open testdata file %s: %v", filename, err)
        }
        originalCBConfig := string(b)

        ast, err := ParseShowFormat(originalCBConfig, configModel)
        if err != nil {
                t.Fatalf("Failed to parse %s: %v", filename, err)
        }

        treesize := ast.TreeSize()
        want := 162
        if treesize != want {
                t.Errorf("Got treesize=%d, want %d", treesize, want)
        }

        newCBConfig, err := WriteShowFormat(ast)
        if err != nil {
                t.Fatalf("Failed calling writeShowFormat: %v", err)
        }

        // Now, the original file has a bunch of version comments in
        // it which we don't decode.  Because of that, we want to
        // compare *without* the comments, so let's strip them here.
        //
        // Also remove completely blank lines.
        lines := []string{}
        for _, line := range strings.SplitAfter(originalCBConfig, "\n") {
                if !strings.HasPrefix(line, "//") && line != "\n" {
                        lines = append(lines, line)
                }
        }
        originalCBConfig = strings.Join(lines, "")

        if newCBConfig != originalCBConfig {
                edits := myers.ComputeEdits("foo", originalCBConfig, newCBConfig)
                fmt.Println(gotextdiff.ToUnified(filename, "output", originalCBConfig, edits))
                t.Errorf("Generated config.show file does not match %s", filename)

        }

}

func TestParseShowToSet(t *testing.T) {
        configModel := getConfigModel(t)
        filename := "testdata/config.show.1"

        b, err := os.ReadFile(filename)
        if err != nil {
                t.Fatalf("Failed to open testdata file: %v", err)
        }
        ast, err := ParseShowFormat(string(b), configModel)
        if err != nil {
                t.Fatalf("Failed to parse %s: %v", filename, err)
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
                t.Fatalf("Failed to open %s: %v", filename, err)
        }

        if set != string(setFile) {
                edits := myers.ComputeEdits("foo", string(setFile), set)
                fmt.Println(gotextdiff.ToUnified(filename, "output", string(setFile), edits))
                t.Errorf("Generated set-format file does not match %s", filename)

        }

}

func TestParseShowDuplication(t *testing.T) {
        configModel := getConfigModel(t)
        config := `firewall {
    flowtable default {
        interface eth1
        offload software
    }
}
firewall {
    flowtable default {
        interface eth2
        offload hardware
    }
}`
        ast, err := ParseShowFormat(config, configModel)
        if err != nil {
                t.Fatalf("Failed to parse static config: %v", err)
        }

        set, err := WriteSetFormat(ast)
        if err != nil {
                t.Fatalf("Failed calling writeSetFormat: %v", err)
        }

        setLines := strings.Split(set, "\n")
        wantLines := 4
        if len(setLines) != wantLines {
                t.Errorf("Got %d lines from WriteSetFormat, want=%d", len(setLines), wantLines)
                fmt.Println(set)
        }
}

func TestParseShowNAT(t *testing.T) {
        configModel := getConfigModel(t)
        showConfig := `nat {
    source {
        rule 100 {
            description "Outbound NAT"
            outbound-interface {
                name eth0
            }
            source {
                address 10.0.0.0/8
            }
            translation {
                address masquerade
            }
        }
    }
}
`
        setConfig := `set nat source rule 100 description 'Outbound NAT'
set nat source rule 100 outbound-interface name 'eth0'
set nat source rule 100 source address '10.0.0.0/8'
set nat source rule 100 translation address 'masquerade'
`

        ast, err := ParseShowFormat(showConfig, configModel)
        if err != nil {
                t.Fatalf("Failed to parse static show config: %v", err)
        }

        set, err := WriteSetFormat(ast)
        if err != nil {
                t.Fatalf("Failed calling writeSetFormat: %v", err)
        }

        if set != setConfig {
                edits := myers.ComputeEdits("foo", set, setConfig)
                fmt.Println(gotextdiff.ToUnified("input", "output", set, edits))
                t.Errorf("Generated set-format file does not match")

        }
}
