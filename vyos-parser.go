package main

import (
	"encoding/xml"
	"encoding/json"
	"fmt"
	"github.com/scottlaird/vyos-parser/xmlreader"
	"io/ioutil"
	"os"
	"path/filepath"
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

	id.Fixup()

	fmt.Printf("********\n")
	id.Print(0)
	fmt.Printf("********\n")
	b, err := json.MarshalIndent(id,"","  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
