package version

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Error types for version parsing failures
var (
	ErrEmptyVersion      = errors.New("version string is empty")
	ErrTooManyComponents = errors.New("version has more than 3 components")
	ErrNonNumeric        = errors.New("version component is not numeric")
	ErrNegativeComponent = errors.New("version component cannot be negative")
	ErrInvalidPrecision  = errors.New("version precision must be 1, 2, or 3")
)

// Version represents a semantic version with Major, Minor, and Patch components.
// Precision indicates how many components are significant (1=Major, 2=Major.Minor, 3=Major.Minor.Patch).
type Version struct {
	Major int `json:"major,omitempty" yaml:"major,omitempty"`
	Minor int `json:"minor,omitempty" yaml:"minor,omitempty"`
	Patch int `json:"patch,omitempty" yaml:"patch,omitempty"`

	// Precision indicates how many components are significant (1, 2, or 3)
	Precision int `json:"precision,omitempty" yaml:"precision,omitempty"`
}

// NewVersion creates a Version with all three components and precision 3.
// Use ParseVersion if you need flexible precision.
func NewVersion(major, minor, patch int) Version {
	return Version{
		Major:     major,
		Minor:     minor,
		Patch:     patch,
		Precision: 3,
	}
}

// String returns the version as a string respecting its precision
func (v Version) String() string {
	switch v.Precision {
	case 1:
		return fmt.Sprintf("%d", v.Major)
	case 2:
		return fmt.Sprintf("%d.%d", v.Major, v.Minor)
	default:
		return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	}
}

// ParseVersion parses a version string in the format "Major", "Major.Minor", "Major.Minor.Patch", or with "v" prefix.
// Returns an error if the version string is invalid.
func ParseVersion(s string) (Version, error) {
	// Check for empty string
	if s == "" {
		return Version{}, ErrEmptyVersion
	}

	// Strip 'v' prefix if present
	s = strings.TrimPrefix(s, "v")
	var v Version

	// Split by dots
	parts := strings.Split(s, ".")
	if len(parts) > 3 {
		return Version{}, ErrTooManyComponents
	}

	// Parse each component
	for i, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return Version{}, fmt.Errorf("%w: %q", ErrNonNumeric, part)
		}
		if num < 0 {
			return Version{}, fmt.Errorf("%w: %d", ErrNegativeComponent, num)
		}

		switch i {
		case 0:
			v.Major = num
		case 1:
			v.Minor = num
		case 2:
			v.Patch = num
		}
	}

	v.Precision = len(parts)
	return v, nil
}

// MustParseVersion parses a version string and panics if parsing fails.
// Useful for initializing package-level constants.
func MustParseVersion(s string) Version {
	v, err := ParseVersion(s)
	if err != nil {
		panic(fmt.Sprintf("MustParseVersion: %v", err))
	}
	return v
}

// EqualsOrNewer returns true if v is equal to or newer than other.
// Only compares components up to the precision of v (e.g., v0.1 matches v0.1.x)
func (v Version) EqualsOrNewer(other Version) bool {
	// Always compare Major
	if v.Major > other.Major {
		return true
	}
	if v.Major < other.Major {
		return false
	}

	// If precision is 1 (Major only), we're equal
	if v.Precision == 1 {
		return true
	}

	// Major versions are equal, compare Minor
	if v.Minor > other.Minor {
		return true
	}
	if v.Minor < other.Minor {
		return false
	}

	// If precision is 2 (Major.Minor), we're equal
	if v.Precision == 2 {
		return true
	}

	// Minor versions are equal, compare Patch
	return v.Patch >= other.Patch
}

// IsNewer returns true if v is strictly newer than other (not equal).
// Respects precision like EqualsOrNewer.
func (v Version) IsNewer(other Version) bool {
	// Always compare Major
	if v.Major > other.Major {
		return true
	}
	if v.Major < other.Major {
		return false
	}

	// If precision is 1 (Major only), they're equal
	if v.Precision == 1 {
		return false
	}

	// Major versions are equal, compare Minor
	if v.Minor > other.Minor {
		return true
	}
	if v.Minor < other.Minor {
		return false
	}

	// If precision is 2 (Major.Minor), they're equal
	if v.Precision == 2 {
		return false
	}

	// Minor versions are equal, compare Patch
	return v.Patch > other.Patch
}

// Equals returns true if v exactly equals other (all components match).
// Unlike EqualsOrNewer, this ignores precision and compares all fields.
func (v Version) Equals(other Version) bool {
	return v.Major == other.Major && v.Minor == other.Minor && v.Patch == other.Patch
}

// Compare returns an integer comparing two versions:
// -1 if v < other, 0 if v == other, 1 if v > other.
// This comparison respects precision like EqualsOrNewer.
// Useful for sorting versions.
func (v Version) Compare(other Version) int {
	// Use lower precision for comparison
	precision := v.Precision
	if other.Precision < precision {
		precision = other.Precision
	}

	// Compare Major
	if v.Major < other.Major {
		return -1
	}
	if v.Major > other.Major {
		return 1
	}

	// Major equal, check if we should compare Minor
	if precision == 1 {
		return 0
	}

	// Compare Minor
	if v.Minor < other.Minor {
		return -1
	}
	if v.Minor > other.Minor {
		return 1
	}

	// Minor equal, check if we should compare Patch
	if precision == 2 {
		return 0
	}

	// Compare Patch
	if v.Patch < other.Patch {
		return -1
	}
	if v.Patch > other.Patch {
		return 1
	}

	return 0
}

// IsValid returns true if the version has valid values.
// All components must be non-negative and precision must be 1, 2, or 3.
func (v Version) IsValid() bool {
	if v.Major < 0 || v.Minor < 0 || v.Patch < 0 {
		return false
	}
	if v.Precision < 1 || v.Precision > 3 {
		return false
	}
	return true
}
