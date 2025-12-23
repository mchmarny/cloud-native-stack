package recommendation

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/version"
)

func TestQueryString(t *testing.T) {
	osv := mustParseVersion(t, "22.04")
	kernel := mustParseVersion(t, "5.15.0")
	k8s := mustParseVersion(t, "1.28.3")

	tests := []struct {
		name  string
		query Query
		want  string
	}{
		{
			name: "all fields",
			query: Query{
				Os:        OSUbuntu,
				OsVersion: osv,
				Kernel:    kernel,
				Service:   ServiceEKS,
				K8s:       k8s,
				GPU:       GPUH100,
				Intent:    IntentTraining,
			},
			want: "OS: ubuntu 22.4, Kernel: 5.15.0, Service: eks, K8s: 1.28.3, GPU: h100, Intent: training",
		},
		{
			name: "overlay subset",
			query: Query{
				Os:      OSUbuntu,
				Service: ServiceEKS,
			},
			want: "OS: ubuntu any, Kernel: any, Service: eks, K8s: any, GPU: any, Intent: any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.query.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestQueryIsMatch(t *testing.T) {
	osv := mustParseVersion(t, "22.04")
	kernel := mustParseVersion(t, "5.15.0")
	k8s := mustParseVersion(t, "1.28.3")

	base := Query{
		Os:        OSUbuntu,
		OsVersion: osv,
		Kernel:    kernel,
		Service:   ServiceEKS,
		K8s:       k8s,
		GPU:       GPUH100,
		Intent:    IntentTraining,
	}

	zeroVersion := version.Version{}

	tests := []struct {
		name  string
		left  Query
		right Query
		want  bool
	}{
		{name: "identical queries", left: base, right: base, want: true},
		{name: "zero value left matches", left: Query{}, right: base, want: true},
		{name: "os mismatch", left: base, right: Query{Os: OSCOS}, want: false},
		{name: "os wildcard left", left: Query{Os: OSAny}, right: base, want: true},
		{name: "os wildcard right", left: base, right: Query{Os: OSAny}, want: true},
		{name: "os version mismatch", left: base, right: Query{OsVersion: mustParseVersion(t, "24.04")}, want: false},
		{name: "os version wildcard left", left: Query{OsVersion: zeroVersion}, right: base, want: true},
		{name: "os version wildcard right", left: base, right: Query{OsVersion: zeroVersion}, want: true},
		{name: "kernel mismatch", left: base, right: Query{Kernel: mustParseVersion(t, "6.0.0")}, want: false},
		{name: "kernel wildcard left", left: Query{Kernel: zeroVersion}, right: base, want: true},
		{name: "kernel wildcard right", left: base, right: Query{Kernel: zeroVersion}, want: true},
		{name: "service mismatch", left: base, right: Query{Service: ServiceGKE}, want: false},
		{name: "service wildcard left", left: Query{Service: ServiceAny}, right: base, want: true},
		{name: "service wildcard right", left: base, right: Query{Service: ServiceAny}, want: true},
		{name: "k8s mismatch", left: base, right: Query{K8s: mustParseVersion(t, "1.29.0")}, want: false},
		{name: "k8s wildcard left", left: Query{K8s: zeroVersion}, right: base, want: true},
		{name: "k8s wildcard right", left: base, right: Query{K8s: zeroVersion}, want: true},
		{name: "gpu mismatch", left: base, right: Query{GPU: GPUB200}, want: false},
		{name: "gpu wildcard left", left: Query{GPU: GPUAny}, right: base, want: true},
		{name: "gpu wildcard right", left: base, right: Query{GPU: GPUAny}, want: true},
		{name: "intent mismatch", left: base, right: Query{Intent: IntentInference}, want: false},
		{name: "intent wildcard left", left: Query{Intent: IntentAny}, right: base, want: true},
		{name: "intent wildcard right", left: base, right: Query{Intent: IntentAny}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := tt.left
			right := tt.right
			if got := left.IsMatch(&right); got != tt.want {
				t.Fatalf("left.IsMatch(right) = %v, want %v", got, tt.want)
			}
			if got := right.IsMatch(&left); got != tt.want {
				t.Fatalf("right.IsMatch(left) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeValue(t *testing.T) {
	type custom string

	tests := []struct {
		name string
		in   custom
		want string
	}{
		{name: "zero value", in: "", want: anyValue},
		{name: "whitespace trimmed", in: "  H100  ", want: "h100"},
		{name: "already normalized", in: "eks", want: "eks"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeValue(tt.in); got != tt.want {
				t.Fatalf("normalizeValue(%q) = %q, want %q", string(tt.in), got, tt.want)
			}
		})
	}
}

func TestNormalizeVersionValue(t *testing.T) {
	v := mustParseVersion(t, "1.2.3")

	tests := []struct {
		name string
		in   version.Version
		want string
	}{
		{name: "zero precision", in: version.Version{}, want: anyValue},
		{name: "semantic version", in: v, want: "1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeVersionValue(tt.in); got != tt.want {
				t.Fatalf("normalizeVersionValue(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseOsFamily(t *testing.T) {
	tests := []struct {
		name    string
		values  url.Values
		want    OsFamily
		wantErr bool
	}{
		{name: "default", values: url.Values{}, want: OSAny},
		{name: "valid", values: url.Values{QueryParamOSFamily: {"ubuntu"}}, want: OSUbuntu},
		{name: "invalid", values: url.Values{QueryParamOSFamily: {"windows"}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOsFamily(tt.values)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseOsFamily() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseServiceType(t *testing.T) {
	tests := []struct {
		name    string
		values  url.Values
		want    ServiceType
		wantErr bool
	}{
		{name: "default", values: url.Values{}, want: ServiceAny},
		{name: "valid", values: url.Values{QueryParamEnvironment: {"eks"}}, want: ServiceEKS},
		{name: "invalid", values: url.Values{QueryParamEnvironment: {"aks"}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseServiceType(tt.values)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseServiceType() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseGPUType(t *testing.T) {
	tests := []struct {
		name    string
		values  url.Values
		want    GPUType
		wantErr bool
	}{
		{name: "default", values: url.Values{}, want: GPUAny},
		{name: "valid", values: url.Values{QueryParamGPU: {"gb200"}}, want: GPUB200},
		{name: "invalid", values: url.Values{QueryParamGPU: {"t4"}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGPUType(tt.values)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseGPUType() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseIntentType(t *testing.T) {
	tests := []struct {
		name    string
		values  url.Values
		want    IntentType
		wantErr bool
	}{
		{name: "default", values: url.Values{}, want: IntentAny},
		{name: "valid", values: url.Values{QueryParamIntent: {"inference"}}, want: IntentInference},
		{name: "invalid", values: url.Values{QueryParamIntent: {"testing"}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIntentType(tt.values)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseIntentType() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseQuery(t *testing.T) {
	osv := mustParseVersion(t, "22.04")
	kernel := mustParseVersion(t, "5.15.0")
	k8s := mustParseVersion(t, "1.28.3")

	tests := []struct {
		name    string
		path    string
		want    *Query
		wantErr bool
	}{
		{
			name: "valid",
			path: "/?os=ubuntu&osv=22.04&kernel=5.15.0&env=eks&k8s=1.28.3&gpu=h100&intent=training",
			want: &Query{
				Os:        OSUbuntu,
				OsVersion: osv,
				Kernel:    kernel,
				Service:   ServiceEKS,
				K8s:       k8s,
				GPU:       GPUH100,
				Intent:    IntentTraining,
			},
		},
		{
			name:    "invalid version",
			path:    "/?osv=bad",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			got, err := ParseQuery(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertQueryEquals(t, tt.want, got)
		})
	}
}

func mustParseVersion(t *testing.T, val string) version.Version {
	t.Helper()
	v, err := version.ParseVersion(val)
	if err != nil {
		t.Fatalf("failed to parse version %q: %v", val, err)
	}
	return v
}

func assertQueryEquals(t *testing.T, want, got *Query) {
	t.Helper()
	if want.Os != got.Os {
		t.Fatalf("Os = %s, want %s", got.Os, want.Os)
	}
	if want.OsVersion != got.OsVersion {
		t.Fatalf("OsVersion = %v, want %v", got.OsVersion, want.OsVersion)
	}
	if want.Kernel != got.Kernel {
		t.Fatalf("Kernel = %v, want %v", got.Kernel, want.Kernel)
	}
	if want.Service != got.Service {
		t.Fatalf("Service = %s, want %s", got.Service, want.Service)
	}
	if want.K8s != got.K8s {
		t.Fatalf("K8s = %v, want %v", got.K8s, want.K8s)
	}
	if want.GPU != got.GPU {
		t.Fatalf("GPU = %s, want %s", got.GPU, want.GPU)
	}
	if want.Intent != got.Intent {
		t.Fatalf("Intent = %s, want %s", got.Intent, want.Intent)
	}
}
