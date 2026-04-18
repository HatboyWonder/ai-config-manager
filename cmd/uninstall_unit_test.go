package cmd

import (
	"strings"
	"testing"

	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/resource"
)

func TestParseResourceArg(t *testing.T) {
	tests := []struct {
		name        string
		arg         string
		wantType    resource.ResourceType
		wantName    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid skill",
			arg:      "skill/my-skill",
			wantType: resource.Skill,
			wantName: "my-skill",
			wantErr:  false,
		},
		{
			name:     "valid command",
			arg:      "command/my-command",
			wantType: resource.Command,
			wantName: "my-command",
			wantErr:  false,
		},
		{
			name:     "valid agent",
			arg:      "agent/my-agent",
			wantType: resource.Agent,
			wantName: "my-agent",
			wantErr:  false,
		},
		{
			name:        "invalid format - no slash",
			arg:         "skill",
			wantErr:     true,
			errContains: "must be 'type/name'",
		},
		{
			name:        "invalid format - empty name",
			arg:         "skill/",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "invalid type",
			arg:         "invalid/name",
			wantErr:     true,
			errContains: "must be one of 'skill', 'command', or 'agent'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotName, err := parseResourceArg(tt.arg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseResourceArg() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseResourceArg() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseResourceArg() unexpected error = %v", err)
				return
			}

			if gotType != tt.wantType {
				t.Errorf("parseResourceArg() type = %v, want %v", gotType, tt.wantType)
			}
			if gotName != tt.wantName {
				t.Errorf("parseResourceArg() name = %v, want %v", gotName, tt.wantName)
			}
		})
	}
}
