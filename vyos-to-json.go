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

	vc := (&configmodel.InterfaceDefinition{}).VyOSConfig()

	// Read through each interface definition file and construct
	// an InterfaceDefintion object based on the XML found in the
	// file.  Then merge them all together into the `vc` object,
	// above.
	for _, file := range files {
		if !file.IsDir() {
			filename := filepath.Join(*interfaceDefinitions, file.Name())
			b, err := os.ReadFile(filename)
			if err != nil {
				panic(err)
			}

			id := &configmodel.InterfaceDefinition{}
			err = xml.Unmarshal(b, &id) // parse XML
			if err != nil {
				panic(err)
			}

			// Convert from configmodel.InterfaceDefinition to configmodel.VyOSConfigNode
			vcn := id.VyOSConfig()

			// And merge the nodes all together.
			vc.Merge(vcn)
		}
	}

	// Finally, write the merged config out as a JSON file.
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
