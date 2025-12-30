package xmp_test

import (
	"strings"
	"testing"

	"sidelight/internal/xmp"
)

func TestMarshal(t *testing.T) {
	settings := xmp.NewCameraRawSettings()
	settings.Exposure2012 = 1.50
	settings.Contrast2012 = 20
	settings.Temperature = 5500
	settings.Tint = 10
	settings.Vibrance = 15

	data, err := xmp.Marshal(settings)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	xmlStr := string(data)

	// Check Header
	if !strings.HasPrefix(xmlStr, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Missing XML header")
	}

	// Check Root Elements and Namespaces
	expectedSubstrings := []string{
		`<x:xmpmeta xmlns:x="adobe:ns:meta/"`,
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">`,
		`<rdf:Description rdf:about=""`,
		`xmlns:crs="http://ns.adobe.com/camera-raw-settings/1.0/"`,
		`crs:ProcessVersion="11.0"`,
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(xmlStr, s) {
			t.Errorf("Expected output to contain %q", s)
		}
	}

	// Check Values
	// Note: XML attribute order is not guaranteed, so we check for individual attributes
	valueChecks := []struct {
		substr string
	}{
		{`crs:Exposure2012="1.5"`},
		{`crs:Contrast2012="20"`},
		{`crs:Temperature="5500"`},
		{`crs:Tint="10"`},
		{`crs:Vibrance="15"`},
	}

	for _, check := range valueChecks {
		if !strings.Contains(xmlStr, check.substr) {
			t.Errorf("Expected output to contain attribute %q", check.substr)
		}
	}

	t.Logf("Generated XML:\n%s", xmlStr)
}

func TestMarshalDefaults(t *testing.T) {
	// Test that even with empty struct (except defaults), we get valid XML
	settings := xmp.NewCameraRawSettings()
	// Force some zeros to see if they are omitted (they should be due to omitempty)
	// Actually, Go zero values:
	// Contrast2012 is int, default 0. Tag has omitempty. So it should NOT appear.
	
	data, err := xmp.Marshal(settings)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	xmlStr := string(data)
	if strings.Contains(xmlStr, `crs:Contrast2012="0"`) {
		t.Error("Expected zero integer value to be omitted")
	}
	
	if !strings.Contains(xmlStr, `crs:ProcessVersion="11.0"`) {
		t.Error("Expected default ProcessVersion")
	}
}
