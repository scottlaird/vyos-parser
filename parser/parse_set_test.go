package parser

import (
        "fmt"
        "os"
        "strings"
        "testing"

        "github.com/hexops/gotextdiff"
        "github.com/hexops/gotextdiff/myers"
        "github.com/scottlaird/vyos-parser/configmodel"
        "github.com/scottlaird/vyos-parser/syntax"
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

func TestParseSetDuplication(t *testing.T) {
        configModel := getConfigModel(t)
        config := `set firewall flowtable default interface eth1
set firewall flowtable default offload software
set firewall flowtable default interface eth0
set firewall flowtable default offload hardware
`
        ast, err := ParseSetFormat(config, configModel)
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

func TestParseSetSort(t *testing.T) {
        configModel := getConfigModel(t)
        config := `set firewall flowtable default interface eth1
set interfaces ethernet eth0 address 'dhcp'
set firewall flowtable default offload software
set firewall flowtable default interface eth0
set firewall flowtable default offload hardware
`
        ast, err := ParseSetFormat(config, configModel)
        if err != nil {
                t.Fatalf("Failed to parse static config: %v", err)
        }

        ast.Sort()

        set, err := WriteSetFormat(ast)
        if err != nil {
                t.Fatalf("Failed calling writeSetFormat: %v", err)
        }

        expected := `set firewall flowtable default interface 'eth0'
set firewall flowtable default interface 'eth1'
set firewall flowtable default offload 'hardware'
set interfaces ethernet eth0 address 'dhcp'
`
        if set != expected {
                edits := myers.ComputeEdits("foo", expected, set)
                fmt.Println(gotextdiff.ToUnified("original", "output", expected, edits))
                t.Errorf("Generated set-format file does not match expected")
        }

}

func TestParseSetNAT(t *testing.T) {
        configModel := getConfigModel(t)
        showConfig := ` nat {
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

        ast, err := ParseSetFormat(setConfig, configModel)
        if err != nil {
                t.Fatalf("Failed to parse static set config: %v", err)
        }

        show, err := WriteShowFormat(ast)
        if err != nil {
                t.Fatalf("Failed calling writeShowFormat: %v", err)
        }

        if show != showConfig {
                edits := myers.ComputeEdits("foo", showConfig, show)
                fmt.Println(gotextdiff.ToUnified("input", "output", showConfig, edits))
                t.Errorf("Generated show-format file does not match")

        }
}
