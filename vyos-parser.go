package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"io/ioutil"
	"path/filepath"
	"github.com/scottlaird/vyos-parser/xmlreader"
)

func main() {
	dir := "interface-definitions"
	files, _ := ioutil.ReadDir(dir)

	id := &xmlreader.InterfaceDefinition{}
	
	for _, file := range files {
		if !file.IsDir() {
			filename := filepath.Join(dir, file.Name())
			b, err := os.ReadFile(filename)
			if err != nil {
				panic(err)
			}
			
			interfacedef := &xmlreader.InterfaceDefinition{}
			
			err = xml.Unmarshal(b, &interfacedef)
			if err != nil {
				panic(err)
			}
			
			//interfacedef.Print(0)

			id.Merge(interfacedef)
		}
	}

	fmt.Printf("********\n")
	id.Print(0)
}

