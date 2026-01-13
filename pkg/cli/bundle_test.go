/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"reflect"
	"testing"
)

func TestParseSetFlags(t *testing.T) {
	tests := []struct {
		name     string
		setFlags []string
		want     map[string]map[string]string
		wantErr  bool
	}{
		{
			name:     "empty flags",
			setFlags: []string{},
			want:     map[string]map[string]string{},
			wantErr:  false,
		},
		{
			name:     "single flag",
			setFlags: []string{"gpuoperator:gds.enabled=true"},
			want: map[string]map[string]string{
				"gpuoperator": {
					"gds.enabled": "true",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple flags same bundler",
			setFlags: []string{
				"gpuoperator:gds.enabled=true",
				"gpuoperator:driver.version=570.86.16",
			},
			want: map[string]map[string]string{
				"gpuoperator": {
					"gds.enabled":    "true",
					"driver.version": "570.86.16",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple flags different bundlers",
			setFlags: []string{
				"gpuoperator:gds.enabled=true",
				"networkoperator:rdma.enabled=true",
			},
			want: map[string]map[string]string{
				"gpuoperator": {
					"gds.enabled": "true",
				},
				"networkoperator": {
					"rdma.enabled": "true",
				},
			},
			wantErr: false,
		},
		{
			name:     "value with equals sign",
			setFlags: []string{"gpuoperator:image.tag=v25.3.0=beta"},
			want: map[string]map[string]string{
				"gpuoperator": {
					"image.tag": "v25.3.0=beta",
				},
			},
			wantErr: false,
		},
		{
			name:     "value with spaces",
			setFlags: []string{"gpuoperator:custom.label=hello world"},
			want: map[string]map[string]string{
				"gpuoperator": {
					"custom.label": "hello world",
				},
			},
			wantErr: false,
		},
		{
			name:     "missing colon",
			setFlags: []string{"gpuoperatorgds.enabled=true"},
			wantErr:  true,
		},
		{
			name:     "missing equals sign",
			setFlags: []string{"gpuoperator:gds.enabledtrue"},
			wantErr:  true,
		},
		{
			name:     "empty path",
			setFlags: []string{"gpuoperator:=true"},
			wantErr:  true,
		},
		{
			name:     "empty value",
			setFlags: []string{"gpuoperator:gds.enabled="},
			wantErr:  true,
		},
		{
			name:     "only bundler name",
			setFlags: []string{"gpuoperator:"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSetFlags(tt.setFlags)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseSetFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSetFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBundleCmd(t *testing.T) {
	cmd := bundleCmd()

	// Verify command configuration
	if cmd.Name != "bundle" {
		t.Errorf("expected command name 'bundle', got %q", cmd.Name)
	}

	// Verify required flags exist
	flagNames := make(map[string]bool)
	for _, flag := range cmd.Flags {
		names := flag.Names()
		for _, name := range names {
			flagNames[name] = true
		}
	}

	requiredFlags := []string{"recipe", "r", "bundlers", "b", "output", "o", "set"}
	for _, flag := range requiredFlags {
		if !flagNames[flag] {
			t.Errorf("expected flag %q to be defined", flag)
		}
	}
}
