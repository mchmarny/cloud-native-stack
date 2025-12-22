package measurement

import (
	"encoding/json"
	"testing"
)

const (
	testSubtypeCluster = "cluster"
	testSubtypeNode    = "node"
	testSubtypePod     = "pod"
)

func TestType_String(t *testing.T) {
	tests := []struct {
		name string
		mt   Type
		want string
	}{
		{"Grub", TypeGrub, "Grub"},
		{"Image", TypeImage, "Image"},
		{"KMod", TypeKMod, "KMod"},
		{"K8s", TypeK8s, "K8s"},
		{"SMI", TypeSMI, "SMI"},
		{"Sysctl", TypeSysctl, "Sysctl"},
		{"SystemD", TypeSystemD, "SystemD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mt.String(); got != tt.want {
				t.Errorf("Type.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseType(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   Type
		wantOk bool
	}{
		{"valid grub", "Grub", TypeGrub, true},
		{"valid k8s", "K8s", TypeK8s, true},
		{"invalid", "Invalid", "", false},
		{"empty", "", "", false},
		{"lowercase", "grub", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := ParseType(tt.input)
			if got != tt.want || gotOk != tt.wantOk {
				t.Errorf("ParseType(%q) = (%v, %v), want (%v, %v)", tt.input, got, gotOk, tt.want, tt.wantOk)
			}
		})
	}
}

func TestToReading(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		wantValue any
		wantType  string
	}{
		{"int", 42, 42, "int"},
		{"int64", int64(9223372036854775807), int64(9223372036854775807), "int64"},
		{"uint", uint(42), uint(42), "uint"},
		{"uint64", uint64(18446744073709551615), uint64(18446744073709551615), "uint64"},
		{"float64", 3.14, 3.14, "float64"},
		{"bool true", true, true, "bool"},
		{"bool false", false, false, "bool"},
		{"string", "hello", "hello", "string"},
		{"empty string", "", "", "string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToReading(tt.value)
			if got == nil {
				t.Fatal("ToReading() returned nil")
			}
			gotValue := got.Any()
			if gotValue != tt.wantValue {
				t.Errorf("ToReading(%v).Any() = %v (%T), want %v (%T)", tt.value, gotValue, gotValue, tt.wantValue, tt.wantValue)
			}
		})
	}
}

func TestScalar_JSON(t *testing.T) {
	tests := []struct {
		name    string
		reading Reading
		want    string
	}{
		{"int", Int(42), "42"},
		{"int64", Int64(9223372036854775807), "9223372036854775807"},
		{"uint", Uint(42), "42"},
		{"uint64", Uint64(18446744073709551615), "18446744073709551615"},
		{"float64", Float64(3.14), "3.14"},
		{"bool true", Bool(true), "true"},
		{"bool false", Bool(false), "false"},
		{"string", Str("hello"), `"hello"`},
		{"empty string", Str(""), `""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.reading)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(data) != tt.want {
				t.Errorf("Marshal() = %v, want %v", string(data), tt.want)
			}
		})
	}
}

func TestScalar_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		reading Reading
		wantVal any
	}{
		{"int", "42", &Scalar[int]{}, 42},
		{"int64", "9223372036854775807", &Scalar[int64]{}, int64(9223372036854775807)},
		{"uint", "42", &Scalar[uint]{}, uint(42)},
		{"uint64", "18446744073709551615", &Scalar[uint64]{}, uint64(18446744073709551615)},
		{"float64", "3.14", &Scalar[float64]{}, float64(3.14)},
		{"bool true", "true", &Scalar[bool]{}, true},
		{"bool false", "false", &Scalar[bool]{}, false},
		{"string", `"hello"`, &Scalar[string]{}, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := json.Unmarshal([]byte(tt.json), tt.reading); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			got := tt.reading.Any()
			if got != tt.wantVal {
				t.Errorf("Unmarshal() value = %v (%T), want %v (%T)", got, got, tt.wantVal, tt.wantVal)
			}
		})
	}
}

func TestMeasurement_Validate(t *testing.T) {
	tests := []struct {
		name    string
		m       *Measurement
		wantErr bool
	}{
		{
			name: "valid measurement",
			m: &Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{
						Name: testSubtypeCluster,
						Data: map[string]Reading{
							"version": Str("1.28.0"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty type",
			m: &Measurement{
				Type: "",
				Subtypes: []Subtype{
					{
						Name: testSubtypeCluster,
						Data: map[string]Reading{
							"version": Str("1.28.0"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "nil subtypes",
			m: &Measurement{
				Type:     TypeK8s,
				Subtypes: nil,
			},
			wantErr: true,
		},
		{
			name: "empty subtypes",
			m: &Measurement{
				Type:     TypeK8s,
				Subtypes: []Subtype{},
			},
			wantErr: true,
		},
		{
			name: "subtype with empty data",
			m: &Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{
						Name: testSubtypeCluster,
						Data: map[string]Reading{},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.m.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMeasurement_GetSubtype(t *testing.T) {
	m := &Measurement{
		Type: TypeK8s,
		Subtypes: []Subtype{
			{
				Name: testSubtypeCluster,
				Data: map[string]Reading{
					"version": Str("1.28.0"),
				},
			},
			{
				Name: testSubtypeNode,
				Data: map[string]Reading{
					"count": Int(3),
				},
			},
		},
	}

	t.Run("existing subtype", func(t *testing.T) {
		st := m.GetSubtype(testSubtypeCluster)
		if st == nil {
			t.Fatal("GetSubtype() returned nil")
		}
		if st.Name != testSubtypeCluster {
			t.Errorf("GetSubtype() name = %v, want cluster", st.Name)
		}
	})

	t.Run("non-existing subtype", func(t *testing.T) {
		st := m.GetSubtype("missing")
		if st != nil {
			t.Errorf("GetSubtype() = %v, want nil", st)
		}
	})
}

func TestMeasurement_HasSubtype(t *testing.T) {
	m := &Measurement{
		Type: TypeK8s,
		Subtypes: []Subtype{
			{Name: testSubtypeCluster, Data: map[string]Reading{"version": Str("1.28.0")}},
			{Name: testSubtypeNode, Data: map[string]Reading{"count": Int(3)}},
		},
	}

	tests := []struct {
		name string
		st   string
		want bool
	}{
		{"existing cluster", testSubtypeCluster, true},
		{"existing node", testSubtypeNode, true},
		{"non-existing", "missing", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.HasSubtype(tt.st); got != tt.want {
				t.Errorf("HasSubtype(%q) = %v, want %v", tt.st, got, tt.want)
			}
		})
	}
}

func TestMeasurement_SubtypeNames(t *testing.T) {
	m := &Measurement{
		Type: TypeK8s,
		Subtypes: []Subtype{
			{Name: testSubtypeCluster, Data: map[string]Reading{"version": Str("1.28.0")}},
			{Name: testSubtypeNode, Data: map[string]Reading{"count": Int(3)}},
			{Name: testSubtypePod, Data: map[string]Reading{"ready": Bool(true)}},
		},
	}

	names := m.SubtypeNames()
	if len(names) != 3 {
		t.Fatalf("SubtypeNames() returned %d names, want 3", len(names))
	}

	expectedNames := []string{testSubtypeCluster, testSubtypeNode, testSubtypePod}
	for i, expected := range expectedNames {
		if names[i] != expected {
			t.Errorf("SubtypeNames()[%d] = %v, want %v", i, names[i], expected)
		}
	}
}

func TestSubtype_Validate(t *testing.T) {
	tests := []struct {
		name    string
		st      *Subtype
		wantErr bool
	}{
		{
			name: "valid subtype",
			st: &Subtype{
				Name: "test",
				Data: map[string]Reading{"key": Str("value")},
			},
			wantErr: false,
		},
		{
			name: "empty data",
			st: &Subtype{
				Name: "test",
				Data: map[string]Reading{},
			},
			wantErr: true,
		},
		{
			name: "nil data",
			st: &Subtype{
				Name: "test",
				Data: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.st.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSubtype_Has(t *testing.T) {
	st := &Subtype{
		Name: "test",
		Data: map[string]Reading{
			"version": Str("1.28.0"),
			"nodes":   Int(3),
		},
	}

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"existing key version", "version", true},
		{"existing key nodes", "nodes", true},
		{"non-existing key", "missing", false},
		{"empty key", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := st.Has(tt.key); got != tt.want {
				t.Errorf("Has(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestSubtype_Get(t *testing.T) {
	st := &Subtype{
		Name: "test",
		Data: map[string]Reading{
			"version": Str("1.28.0"),
		},
	}

	t.Run("existing key", func(t *testing.T) {
		got := st.Get("version")
		if got == nil {
			t.Fatal("Get() returned nil")
		}
		if v, ok := got.Any().(string); !ok || v != "1.28.0" {
			t.Errorf("Get() = %v, want 1.28.0", got.Any())
		}
	})

	t.Run("non-existing key", func(t *testing.T) {
		got := st.Get("missing")
		if got != nil {
			t.Errorf("Get() = %v, want nil", got)
		}
	})
}

func TestSubtype_Keys(t *testing.T) {
	st := &Subtype{
		Name: "test",
		Data: map[string]Reading{
			"version": Str("1.28.0"),
			"nodes":   Int(3),
			"ready":   Bool(true),
		},
	}

	keys := st.Keys()
	if len(keys) != 3 {
		t.Fatalf("Keys() returned %d keys, want 3", len(keys))
	}

	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	expectedKeys := []string{"version", "nodes", "ready"}
	for _, k := range expectedKeys {
		if !keyMap[k] {
			t.Errorf("Keys() missing key %q", k)
		}
	}
}

func TestSubtype_GetString(t *testing.T) {
	st := &Subtype{
		Name: "test",
		Data: map[string]Reading{
			"version": Str("1.28.0"),
			"nodes":   Int(3),
		},
	}

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr bool
	}{
		{"valid string", "version", "1.28.0", false},
		{"wrong type", "nodes", "", true},
		{"missing key", "missing", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := st.GetString(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetString(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetString(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestSubtype_GetInt64(t *testing.T) {
	st := &Subtype{
		Name: "test",
		Data: map[string]Reading{
			"int_value":   Int(42),
			"int64_value": Int64(9223372036854775807),
			"version":     Str("1.28.0"),
		},
	}

	tests := []struct {
		name    string
		key     string
		want    int64
		wantErr bool
	}{
		{"int value", "int_value", 42, false},
		{"int64 value", "int64_value", 9223372036854775807, false},
		{"wrong type", "version", 0, true},
		{"missing key", "missing", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := st.GetInt64(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInt64(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetInt64(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestSubtype_GetUint64(t *testing.T) {
	st := &Subtype{
		Name: "test",
		Data: map[string]Reading{
			"uint_value":   Uint(42),
			"uint64_value": Uint64(18446744073709551615),
			"version":      Str("1.0.0"),
		},
	}

	tests := []struct {
		name    string
		key     string
		want    uint64
		wantErr bool
	}{
		{"uint value", "uint_value", 42, false},
		{"uint64 value", "uint64_value", 18446744073709551615, false},
		{"wrong type", "version", 0, true},
		{"missing key", "missing", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := st.GetUint64(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUint64(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetUint64(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestSubtype_GetFloat64(t *testing.T) {
	st := &Subtype{
		Name: "test",
		Data: map[string]Reading{
			"temperature": Float64(82.5),
			"version":     Str("1.0.0"),
		},
	}

	tests := []struct {
		name    string
		key     string
		want    float64
		wantErr bool
	}{
		{"valid float64", "temperature", 82.5, false},
		{"wrong type", "version", 0, true},
		{"missing key", "missing", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := st.GetFloat64(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFloat64(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetFloat64(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestSubtype_GetBool(t *testing.T) {
	st := &Subtype{
		Name: "test",
		Data: map[string]Reading{
			"ready":   Bool(true),
			"stopped": Bool(false),
			"version": Str("1.0.0"),
		},
	}

	tests := []struct {
		name    string
		key     string
		want    bool
		wantErr bool
	}{
		{"true value", "ready", true, false},
		{"false value", "stopped", false, false},
		{"wrong type", "version", false, true},
		{"missing key", "missing", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := st.GetBool(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBool(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBool(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestMeasurement_JSON(t *testing.T) {
	original := &Measurement{
		Type: TypeK8s,
		Subtypes: []Subtype{
			{
				Name: testSubtypeCluster,
				Data: map[string]Reading{
					"version": Str("1.28.0"),
					"nodes":   Int(3),
					"ready":   Bool(true),
					"cpu":     Float64(85.5),
				},
			},
		},
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Verify JSON structure
	var jsonData map[string]any
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Fatalf("Unmarshal to map error = %v", err)
	}

	// Verify basic fields in JSON
	if jsonData["type"] != string(original.Type) {
		t.Errorf("JSON type = %v, want %v", jsonData["type"], original.Type)
	}

	// Verify subtypes field exists
	subtypes, ok := jsonData["subtypes"].([]any)
	if !ok {
		t.Fatalf("JSON subtypes is not an array")
	}
	if len(subtypes) != len(original.Subtypes) {
		t.Errorf("JSON subtypes length = %d, want %d", len(subtypes), len(original.Subtypes))
	}

	// Verify first subtype
	if len(subtypes) > 0 {
		st, ok := subtypes[0].(map[string]any)
		if !ok {
			t.Fatalf("JSON subtype[0] is not a map")
		}
		if st["subtype"] != testSubtypeCluster {
			t.Errorf("JSON subtype[0].subtype = %v, want cluster", st["subtype"])
		}

		dataMap, ok := st["data"].(map[string]any)
		if !ok {
			t.Fatalf("JSON subtype[0].data is not a map")
		}
		if dataMap["version"] != "1.28.0" {
			t.Errorf("JSON subtype[0].data.version = %v, want 1.28.0", dataMap["version"])
		}
	}
}

func TestConvenienceConstructors(t *testing.T) {
	tests := []struct {
		name    string
		reading Reading
		wantVal any
	}{
		{"Int", Int(42), 42},
		{"Int64", Int64(9223372036854775807), int64(9223372036854775807)},
		{"Uint", Uint(42), uint(42)},
		{"Uint64", Uint64(18446744073709551615), uint64(18446744073709551615)},
		{"Float64", Float64(3.14159), float64(3.14159)},
		{"Bool true", Bool(true), true},
		{"Bool false", Bool(false), false},
		{"Str", Str("hello world"), "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.reading.Any()
			if got != tt.wantVal {
				t.Errorf("Any() = %v (%T), want %v (%T)", got, got, tt.wantVal, tt.wantVal)
			}

			// Verify it implements Reading interface
			tt.reading.isReading()

			// Verify it can be marshaled
			_, err := json.Marshal(tt.reading)
			if err != nil {
				t.Errorf("Marshal() error = %v", err)
			}
		})
	}
}
