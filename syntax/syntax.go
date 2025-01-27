package syntax

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/scottlaird/vyos-parser/configmodel"
)

//go:embed *.json.gz
//go:embed default.version
var f embed.FS

func GetDefaultVersion() (string, error) {
	data, err := f.ReadFile("default.version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func GetDefaultConfigModel() (*configmodel.VyOSConfigNode, error) {
	version, err := GetDefaultVersion()
	if err != nil {
		return nil, err
	}
	return GetConfigModel(version)
}

func GetConfigModel(name string) (*configmodel.VyOSConfigNode, error) {
	data, err := f.ReadFile(name + ".json.gz")
	if err != nil {
		return nil, fmt.Errorf("Could not open config model for %q: %v", name, err)
	}
	byteReader := bytes.NewReader(data)

	gzipReader, err := gzip.NewReader(byteReader)
	if err != nil {
		return nil, fmt.Errorf("Could not open gzip decompressor for %q: %v", name, err)
	}
	defer gzipReader.Close()
	jsonData, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("Could not read gzip data for %q: %v", name, err)
	}

	vc := &configmodel.VyOSConfigNode{}
	err = json.Unmarshal(jsonData, &vc)

	if err != nil {
		return nil, fmt.Errorf("Could not read open gzip decompressor for %q: %v", name, err)
	}
	return vc, nil
}
