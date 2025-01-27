package syntax

import (
	"testing"
)


func TestGetDefaultVersion(t *testing.T) {
	version, err := GetDefaultVersion()
	if err != nil {
		t.Fatalf("Got error from GetDefaultVersion(): %v", err)
	}
	if len(version) != 13 {
		t.Errorf("Got len(version)=%d, want 13 (%q)", len(version), version)
	}
}

func TestGetDefaultConfigModel(t *testing.T) {
	vc, err := GetDefaultConfigModel()
	if err != nil {
		t.Fatalf("Got error from GetDefaultConfigModel(): %v", err)
	}

	if vc==nil {
		t.Fatal("Got nil from GetDefaultConfigModel(), wanted *VyOSCOnfigNode")
	}
	// Should probably check that the config is actually sane at some point...
}
