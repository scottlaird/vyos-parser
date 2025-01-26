package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/scottlaird/vyos-parser/configmodel"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	interfaceDefinitions = flag.String("interface-definitions", "vyos-1x/build/interface-definitions", "Source directory for interface XML files")
	out                  = flag.String("out", "", "Output file")
)

func main() {
	flag.Parse()
	files, _ := ioutil.ReadDir(*interfaceDefinitions)

	id := &configmodel.InterfaceDefinition{}
	vc := id.VyOSConfig()

	for _, file := range files {
		if !file.IsDir() {
			filename := filepath.Join(*interfaceDefinitions, file.Name())
			b, err := os.ReadFile(filename)
			if err != nil {
				panic(err)
			}

			interfacedef := &configmodel.InterfaceDefinition{}
			err = xml.Unmarshal(b, &interfacedef)
			if err != nil {
				panic(err)
			}

			vcn := interfacedef.VyOSConfig()
			vc.Merge(vcn)
		}
	}

	id.Fixup()

	b, err := json.MarshalIndent(vc, "", "  ")
	if err != nil {
		panic(err)
	}

	if out == nil || *out == "" {
		fmt.Println(string(b))
	} else {
		err = os.WriteFile(*out, b, 0644)
		if err != nil {
			panic(err)
		}
	}
}
