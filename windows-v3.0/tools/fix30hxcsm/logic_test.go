package main

import (
	"testing"
)

func TestParseDriverClassFromEnum(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{`{4d36e968-e325-11ce-bfc1-08002be10318}\0001`, "0001"},
		{`{4D36E968-E325-11CE-BFC1-08002BE10318}\0000`, "0000"},
		{`{4d36e968-e325-11ce-bfc1-08002be10318}\0012`, "0012"},
		{"invalid_string", ""},
		{"", ""},
	}

	for _, c := range cases {
		got := ParseDriverClassIndex(c.input)
		if got != c.expected {
			t.Errorf("ParseDriverClassIndex(%q) = %q, expected %q", c.input, got, c.expected)
		}
	}
}

func TestIsGenericBasicDisplayDriver(t *testing.T) {
	cases := []struct {
		desc     string
		provider string
		expected bool
	}{
		{"Microsoft Basic Display Adapter", "Microsoft", true},
		{"Microsoft Basic Display Driver", "Microsoft Corporation", true},
		{"Standard VGA Graphics Adapter", "Microsoft", true},
		{"NVIDIA CMP 30HX", "NVIDIA", false},
		{"NVIDIA GeForce GTX 1660 SUPER", "NVIDIA", false},
		{"Intel(R) UHD Graphics 630", "Intel Corporation", false},
	}

	for _, c := range cases {
		got := IsGenericBasicDisplayDriver(c.desc, c.provider)
		if got != c.expected {
			t.Errorf("IsGenericBasicDisplayDriver(%q, %q) = %v, expected %v", c.desc, c.provider, got, c.expected)
		}
	}
}
