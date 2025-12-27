package version

import (
	"testing"
)

// FuzzParseVersion performs fuzz testing on ParseVersion to find edge cases
func FuzzParseVersion(f *testing.F) {
	// Seed corpus with valid and edge case inputs
	f.Add("1")
	f.Add("v1")
	f.Add("1.2")
	f.Add("v1.2")
	f.Add("1.2.3")
	f.Add("v1.2.3")
	f.Add("0")
	f.Add("0.0")
	f.Add("0.0.0")
	f.Add("999.999.999")
	f.Add("")
	f.Add(".")
	f.Add("..")
	f.Add("1.")
	f.Add(".1")
	f.Add("1..2")
	f.Add("v")
	f.Add("vv1")
	f.Add("-1")
	f.Add("1.-2")
	f.Add("a.b.c")
	f.Add("1.2.3.4")
	f.Add("1.2.3.4.5")
	f.Add("   1.2.3")
	f.Add("1.2.3   ")
	f.Add("1. 2.3")

	f.Fuzz(func(t *testing.T, input string) {
		// ParseVersion should never panic
		v, err := ParseVersion(input)

		// If parsing succeeded, verify the version is valid
		if err == nil {
			// Version should be valid
			if !v.IsValid() {
				t.Errorf("ParseVersion(%q) returned invalid version: %+v", input, v)
			}

			// String() should not panic
			s := v.String()

			// Re-parsing the string should produce the same version
			v2, err2 := ParseVersion(s)
			if err2 != nil {
				t.Errorf("Re-parsing %q (from %q) failed: %v", s, input, err2)
			} else if v.Major != v2.Major || v.Minor != v2.Minor || v.Patch != v2.Patch || v.Precision != v2.Precision {
				t.Errorf("Round-trip mismatch for %q: %+v != %+v", input, v, v2)
			}

			// All version components should be non-negative
			if v.Major < 0 || v.Minor < 0 || v.Patch < 0 {
				t.Errorf("ParseVersion(%q) returned negative component: %+v", input, v)
			}

			// Precision should be 1, 2, or 3
			if v.Precision < 1 || v.Precision > 3 {
				t.Errorf("ParseVersion(%q) returned invalid precision: %d", input, v.Precision)
			}

			// Test comparison methods don't panic
			v3 := NewVersion(1, 2, 3)
			_ = v.EqualsOrNewer(v3)
			_ = v.IsNewer(v3)
			_ = v.Equals(v3)
			_ = v.Compare(v3)
		}
	})
}
