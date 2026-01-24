package recipe

import (
	"net/http"
	"net/url"
	"testing"
)

func TestParseCriteriaServiceType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CriteriaServiceType
		wantErr bool
	}{
		{"empty", "", CriteriaServiceAny, false},
		{"any", "any", CriteriaServiceAny, false},
		{"eks", "eks", CriteriaServiceEKS, false},
		{"EKS uppercase", "EKS", CriteriaServiceEKS, false},
		{"gke", "gke", CriteriaServiceGKE, false},
		{"aks", "aks", CriteriaServiceAKS, false},
		{"oke", "oke", CriteriaServiceOKE, false},
		{"self-managed", "self-managed", CriteriaServiceAny, false},
		{"self", "self", CriteriaServiceAny, false},
		{"vanilla", "vanilla", CriteriaServiceAny, false},
		{"invalid", "invalid", CriteriaServiceAny, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCriteriaServiceType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCriteriaServiceType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseCriteriaServiceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCriteriaAcceleratorType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CriteriaAcceleratorType
		wantErr bool
	}{
		{"empty", "", CriteriaAcceleratorAny, false},
		{"any", "any", CriteriaAcceleratorAny, false},
		{"h100", "h100", CriteriaAcceleratorH100, false},
		{"H100 uppercase", "H100", CriteriaAcceleratorH100, false},
		{"gb200", "gb200", CriteriaAcceleratorGB200, false},
		{"a100", "a100", CriteriaAcceleratorA100, false},
		{"l40", "l40", CriteriaAcceleratorL40, false},
		{"invalid", "v100", CriteriaAcceleratorAny, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCriteriaAcceleratorType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCriteriaAcceleratorType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseCriteriaAcceleratorType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCriteriaIntentType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CriteriaIntentType
		wantErr bool
	}{
		{"empty", "", CriteriaIntentAny, false},
		{"any", "any", CriteriaIntentAny, false},
		{"training", "training", CriteriaIntentTraining, false},
		{"inference", "inference", CriteriaIntentInference, false},
		{"invalid", "serving", CriteriaIntentAny, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCriteriaIntentType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCriteriaIntentType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseCriteriaIntentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCriteriaMatches(t *testing.T) {
	tests := []struct {
		name     string
		criteria *Criteria
		other    *Criteria
		want     bool
	}{
		{
			name:     "nil other",
			criteria: NewCriteria(),
			other:    nil,
			want:     true,
		},
		{
			name:     "all any matches all any",
			criteria: NewCriteria(),
			other:    NewCriteria(),
			want:     true,
		},
		{
			name: "specific recipe does not match generic query",
			criteria: &Criteria{
				Service: CriteriaServiceEKS,
			},
			other: NewCriteria(),
			want:  false, // Query "any" only matches generic recipes
		},
		{
			name:     "generic recipe matches specific query",
			criteria: NewCriteria(), // Recipe: all "any"
			other: &Criteria{
				Service: CriteriaServiceEKS,
			},
			want: true, // Recipe is generic, matches any query value
		},
		{
			name: "same service matches",
			criteria: &Criteria{
				Service: CriteriaServiceEKS,
			},
			other: &Criteria{
				Service: CriteriaServiceEKS,
			},
			want: true,
		},
		{
			name: "different service does not match",
			criteria: &Criteria{
				Service: CriteriaServiceEKS,
			},
			other: &Criteria{
				Service: CriteriaServiceGKE,
			},
			want: false,
		},
		{
			name: "partial match on multiple fields",
			criteria: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorH100,
				Intent:      CriteriaIntentTraining,
			},
			other: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorH100,
				Intent:      CriteriaIntentTraining,
			},
			want: true,
		},
		{
			name: "one field mismatch",
			criteria: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorH100,
				Intent:      CriteriaIntentTraining,
			},
			other: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorGB200,
				Intent:      CriteriaIntentTraining,
			},
			want: false,
		},
		{
			name: "recipe with partial criteria matches query with more fields",
			criteria: &Criteria{
				Service: CriteriaServiceEKS,
				// Accelerator is "any" (zero value)
			},
			other: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorGB200,
			},
			want: true, // Recipe service=eks matches, accelerator is generic so matches any
		},
		{
			name: "recipe with more specific criteria does not match less specific query",
			criteria: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorGB200,
			},
			other: &Criteria{
				Service: CriteriaServiceEKS,
				// Accelerator is "any"
			},
			want: false, // Query doesn't specify accelerator, but recipe requires gb200
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.criteria.Matches(tt.other); got != tt.want {
				t.Errorf("Criteria.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCriteriaSpecificity(t *testing.T) {
	tests := []struct {
		name     string
		criteria *Criteria
		want     int
	}{
		{
			name:     "all any",
			criteria: NewCriteria(),
			want:     0,
		},
		{
			name: "one field",
			criteria: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorAny,
				Intent:      CriteriaIntentAny,
				OS:          CriteriaOSAny,
				Nodes:       0,
			},
			want: 1,
		},
		{
			name: "three fields",
			criteria: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorH100,
				Intent:      CriteriaIntentTraining,
				OS:          CriteriaOSAny,
				Nodes:       0,
			},
			want: 3,
		},
		{
			name: "all fields",
			criteria: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorH100,
				Intent:      CriteriaIntentTraining,
				OS:          CriteriaOSUbuntu,
				Nodes:       100,
			},
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.criteria.Specificity(); got != tt.want {
				t.Errorf("Criteria.Specificity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildCriteria(t *testing.T) {
	tests := []struct {
		name    string
		opts    []CriteriaOption
		want    *Criteria
		wantErr bool
	}{
		{
			name: "no options",
			opts: nil,
			want: NewCriteria(),
		},
		{
			name: "with service",
			opts: []CriteriaOption{WithCriteriaService("eks")},
			want: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorAny,
				Intent:      CriteriaIntentAny,
				OS:          CriteriaOSAny,
			},
		},
		{
			name: "with multiple options",
			opts: []CriteriaOption{
				WithCriteriaService("eks"),
				WithCriteriaAccelerator("h100"),
				WithCriteriaIntent("training"),
			},
			want: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorH100,
				Intent:      CriteriaIntentTraining,
				OS:          CriteriaOSAny,
			},
		},
		{
			name:    "invalid service",
			opts:    []CriteriaOption{WithCriteriaService("invalid")},
			wantErr: true,
		},
		{
			name:    "invalid accelerator",
			opts:    []CriteriaOption{WithCriteriaAccelerator("v100")},
			wantErr: true,
		},
		{
			name:    "negative nodes",
			opts:    []CriteriaOption{WithCriteriaNodes(-1)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildCriteria(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildCriteria() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Service != tt.want.Service ||
				got.Accelerator != tt.want.Accelerator ||
				got.Intent != tt.want.Intent {

				t.Errorf("BuildCriteria() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCriteriaFromValues(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    *Criteria
		wantErr bool
	}{
		{
			name:  "empty query defaults to any",
			query: "",
			want: &Criteria{
				Service:     CriteriaServiceAny,
				Accelerator: CriteriaAcceleratorAny,
				Intent:      CriteriaIntentAny,
				OS:          CriteriaOSAny,
				Nodes:       0,
			},
			wantErr: false,
		},
		{
			name:  "all parameters",
			query: "service=eks&accelerator=h100&intent=training&os=ubuntu&nodes=8",
			want: &Criteria{
				Service:     CriteriaServiceEKS,
				Accelerator: CriteriaAcceleratorH100,
				Intent:      CriteriaIntentTraining,
				OS:          CriteriaOSUbuntu,
				Nodes:       8,
			},
			wantErr: false,
		},
		{
			name:  "gpu alias for accelerator",
			query: "gpu=gb200",
			want: &Criteria{
				Service:     CriteriaServiceAny,
				Accelerator: CriteriaAcceleratorGB200,
				Intent:      CriteriaIntentAny,
				OS:          CriteriaOSAny,
				Nodes:       0,
			},
			wantErr: false,
		},
		{
			name:  "accelerator takes precedence over gpu",
			query: "accelerator=h100&gpu=a100",
			want: &Criteria{
				Service:     CriteriaServiceAny,
				Accelerator: CriteriaAcceleratorH100,
				Intent:      CriteriaIntentAny,
				OS:          CriteriaOSAny,
				Nodes:       0,
			},
			wantErr: false,
		},
		{
			name:    "invalid service",
			query:   "service=invalid",
			wantErr: true,
		},
		{
			name:    "invalid accelerator",
			query:   "accelerator=invalid",
			wantErr: true,
		},
		{
			name:    "invalid intent",
			query:   "intent=invalid",
			wantErr: true,
		},
		{
			name:    "invalid os",
			query:   "os=invalid",
			wantErr: true,
		},
		{
			query:   "os=invalid",
			wantErr: true,
		},
		{
			name:    "invalid nodes - not a number",
			query:   "nodes=abc",
			wantErr: true,
		},
		{
			name:    "invalid nodes - negative",
			query:   "nodes=-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := parseQuery(tt.query)

			got, err := ParseCriteriaFromValues(values)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCriteriaFromValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Service != tt.want.Service {
				t.Errorf("Service = %v, want %v", got.Service, tt.want.Service)
			}
			if got.Accelerator != tt.want.Accelerator {
				t.Errorf("Accelerator = %v, want %v", got.Accelerator, tt.want.Accelerator)
			}
			if got.Intent != tt.want.Intent {
				t.Errorf("Intent = %v, want %v", got.Intent, tt.want.Intent)
			}
			if got.OS != tt.want.OS {
				t.Errorf("OS = %v, want %v", got.OS, tt.want.OS)
			}
			if got.OS != tt.want.OS {
				t.Errorf("OS = %v, want %v", got.OS, tt.want.OS)
			}
			if got.Nodes != tt.want.Nodes {
				t.Errorf("Nodes = %v, want %v", got.Nodes, tt.want.Nodes)
			}
		})
	}
}

func TestParseCriteriaFromRequest(t *testing.T) {
	t.Run("nil request returns error", func(t *testing.T) {
		_, err := ParseCriteriaFromRequest(nil)
		if err == nil {
			t.Error("expected error for nil request")
		}
	})

	t.Run("valid request", func(t *testing.T) {
		req := createTestRequest("service=gke&accelerator=a100")
		got, err := ParseCriteriaFromRequest(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Service != CriteriaServiceGKE {
			t.Errorf("Service = %v, want %v", got.Service, CriteriaServiceGKE)
		}
		if got.Accelerator != CriteriaAcceleratorA100 {
			t.Errorf("Accelerator = %v, want %v", got.Accelerator, CriteriaAcceleratorA100)
		}
	})
}

// parseQuery is a helper to parse URL query strings for testing.
func parseQuery(query string) map[string][]string {
	values := make(map[string][]string)
	if query == "" {
		return values
	}
	for _, pair := range splitQueryParams(query) {
		parts := splitQueryParam(pair)
		if len(parts) == 2 {
			values[parts[0]] = append(values[parts[0]], parts[1])
		}
	}
	return values
}

// splitQueryParams splits a query string on &.
func splitQueryParams(query string) []string {
	result := []string{}
	start := 0
	for i, c := range query {
		if c == '&' {
			if i > start {
				result = append(result, query[start:i])
			}
			start = i + 1
		}
	}
	if start < len(query) {
		result = append(result, query[start:])
	}
	return result
}

// splitQueryParam splits a query param on =.
func splitQueryParam(param string) []string {
	for i, c := range param {
		if c == '=' {
			return []string{param[:i], param[i+1:]}
		}
	}
	return []string{param}
}

// createTestRequest creates a test HTTP request with given query params.
func createTestRequest(query string) *http.Request {
	req := &http.Request{}
	if query != "" {
		req.URL = &url.URL{RawQuery: query}
	} else {
		req.URL = &url.URL{}
	}
	return req
}

func TestGetCriteriaServiceTypes(t *testing.T) {
	types := GetCriteriaServiceTypes()

	// Should return sorted list
	expected := []string{"aks", "eks", "gke", "oke"}
	if len(types) != len(expected) {
		t.Errorf("GetCriteriaServiceTypes() returned %d types, want %d", len(types), len(expected))
	}

	for i, exp := range expected {
		if types[i] != exp {
			t.Errorf("GetCriteriaServiceTypes()[%d] = %s, want %s", i, types[i], exp)
		}
	}

	// Verify each type can be parsed
	for _, st := range types {
		_, err := ParseCriteriaServiceType(st)
		if err != nil {
			t.Errorf("ParseCriteriaServiceType(%s) error = %v", st, err)
		}
	}
}

func TestGetCriteriaAcceleratorTypes(t *testing.T) {
	types := GetCriteriaAcceleratorTypes()

	// Should return sorted list
	expected := []string{"a100", "gb200", "h100", "l40"}
	if len(types) != len(expected) {
		t.Errorf("GetCriteriaAcceleratorTypes() returned %d types, want %d", len(types), len(expected))
	}

	for i, exp := range expected {
		if types[i] != exp {
			t.Errorf("GetCriteriaAcceleratorTypes()[%d] = %s, want %s", i, types[i], exp)
		}
	}

	// Verify each type can be parsed
	for _, at := range types {
		_, err := ParseCriteriaAcceleratorType(at)
		if err != nil {
			t.Errorf("ParseCriteriaAcceleratorType(%s) error = %v", at, err)
		}
	}
}

func TestGetCriteriaIntentTypes(t *testing.T) {
	types := GetCriteriaIntentTypes()

	// Should return sorted list
	expected := []string{"inference", "training"}
	if len(types) != len(expected) {
		t.Errorf("GetCriteriaIntentTypes() returned %d types, want %d", len(types), len(expected))
	}

	for i, exp := range expected {
		if types[i] != exp {
			t.Errorf("GetCriteriaIntentTypes()[%d] = %s, want %s", i, types[i], exp)
		}
	}

	// Verify each type can be parsed
	for _, it := range types {
		_, err := ParseCriteriaIntentType(it)
		if err != nil {
			t.Errorf("ParseCriteriaIntentType(%s) error = %v", it, err)
		}
	}
}

func TestGetCriteriaOSTypes(t *testing.T) {
	types := GetCriteriaOSTypes()

	// Should return sorted list
	expected := []string{"amazonlinux", "cos", "rhel", "ubuntu"}
	if len(types) != len(expected) {
		t.Errorf("GetCriteriaOSTypes() returned %d types, want %d", len(types), len(expected))
	}

	for i, exp := range expected {
		if types[i] != exp {
			t.Errorf("GetCriteriaOSTypes()[%d] = %s, want %s", i, types[i], exp)
		}
	}

	// Verify each type can be parsed
	for _, ot := range types {
		_, err := ParseCriteriaOSType(ot)
		if err != nil {
			t.Errorf("ParseCriteriaOSType(%s) error = %v", ot, err)
		}
	}
}

func TestParseCriteriaOSType_AllAliases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  CriteriaOSType
	}{
		{"amazonlinux", "amazonlinux", CriteriaOSAmazonLinux},
		{"al2", "al2", CriteriaOSAmazonLinux},
		{"al2023", "al2023", CriteriaOSAmazonLinux},
		{"ubuntu", "ubuntu", CriteriaOSUbuntu},
		{"rhel", "rhel", CriteriaOSRHEL},
		{"cos", "cos", CriteriaOSCOS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCriteriaOSType(tt.input)
			if err != nil {
				t.Errorf("ParseCriteriaOSType(%s) error = %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseCriteriaOSType(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
