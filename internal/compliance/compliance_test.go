package compliance

import (
	"testing"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()

	if len(reg.rules) == 0 {
		t.Fatal("expected rules to be registered")
	}
	if len(reg.frameworks) == 0 {
		t.Fatal("expected frameworks to be registered")
	}
}

func TestListFrameworks(t *testing.T) {
	reg := NewRegistry()
	frameworks := reg.ListFrameworks()

	if len(frameworks) == 0 {
		t.Fatal("expected at least one framework")
	}

	found := false
	for _, fw := range frameworks {
		if fw.ID == FrameworkCISGCP {
			found = true
			if fw.Version != "4.0.0" {
				t.Errorf("expected CIS GCP version 4.0.0, got %s", fw.Version)
			}
			if fw.Provider != domain.ProviderGCP {
				t.Errorf("expected GCP provider, got %s", fw.Provider)
			}
		}
	}
	if !found {
		t.Error("expected to find CIS GCP framework")
	}
}

func TestListFrameworksForProvider(t *testing.T) {
	reg := NewRegistry()

	gcpFrameworks := reg.ListFrameworksForProvider(domain.ProviderGCP)
	if len(gcpFrameworks) == 0 {
		t.Fatal("expected GCP frameworks")
	}

	awsFrameworks := reg.ListFrameworksForProvider(domain.ProviderAWS)
	if len(awsFrameworks) != 0 {
		t.Errorf("expected no AWS frameworks yet, got %d", len(awsFrameworks))
	}
}

func TestValidateFrameworks(t *testing.T) {
	reg := NewRegistry()

	tests := []struct {
		name    string
		ids     []string
		wantErr bool
	}{
		{"valid CIS GCP", []string{"cis_gcp_v4"}, false},
		{"empty list", []string{}, false},
		{"invalid framework", []string{"hipaa"}, true},
		{"mix of valid and invalid", []string{"cis_gcp_v4", "nonexistent"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reg.ValidateFrameworks(tt.ids)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFrameworks(%v) error = %v, wantErr %v", tt.ids, err, tt.wantErr)
			}
		})
	}
}

func TestGetRules(t *testing.T) {
	reg := NewRegistry()

	rules := reg.GetRules([]string{"cis_gcp_v4"})
	if len(rules) == 0 {
		t.Fatal("expected CIS GCP rules")
	}

	for _, rule := range rules {
		if rule.Framework != FrameworkCISGCP {
			t.Errorf("rule %s has wrong framework: %s", rule.ID, rule.Framework)
		}
	}

	// Empty frameworks returns nothing
	empty := reg.GetRules([]string{})
	if len(empty) != 0 {
		t.Errorf("expected 0 rules for empty frameworks, got %d", len(empty))
	}
}

func TestGetRulesForResource(t *testing.T) {
	reg := NewRegistry()

	tests := []struct {
		name        string
		frameworks  []string
		provider    domain.CloudProvider
		kind        domain.ResourceKind
		serviceName string
		wantMin     int // minimum expected rules
	}{
		{
			name:        "database rules for Cloud SQL",
			frameworks:  []string{"cis_gcp_v4"},
			provider:    domain.ProviderGCP,
			kind:        domain.ResourceDatabase,
			serviceName: "Cloud SQL",
			wantMin:     5, // 6.4, 6.5, 6.6, 6.7, plus PG-specific rules
		},
		{
			name:        "compute rules for Compute Engine",
			frameworks:  []string{"cis_gcp_v4"},
			provider:    domain.ProviderGCP,
			kind:        domain.ResourceCompute,
			serviceName: "Compute Engine",
			wantMin:     5,
		},
		{
			name:        "storage rules",
			frameworks:  []string{"cis_gcp_v4"},
			provider:    domain.ProviderGCP,
			kind:        domain.ResourceStorage,
			serviceName: "Cloud Storage",
			wantMin:     2, // 5.1, 5.2
		},
		{
			name:        "network rules",
			frameworks:  []string{"cis_gcp_v4"},
			provider:    domain.ProviderGCP,
			kind:        domain.ResourceNetwork,
			serviceName: "VPC",
			wantMin:     3,
		},
		{
			name:        "no rules for AWS (not yet registered)",
			frameworks:  []string{"cis_gcp_v4"},
			provider:    domain.ProviderAWS,
			kind:        domain.ResourceDatabase,
			serviceName: "RDS",
			wantMin:     0,
		},
		{
			name:        "no rules for empty frameworks",
			frameworks:  []string{},
			provider:    domain.ProviderGCP,
			kind:        domain.ResourceDatabase,
			serviceName: "Cloud SQL",
			wantMin:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := reg.GetRulesForResource(tt.frameworks, tt.provider, tt.kind, tt.serviceName)
			if len(rules) < tt.wantMin {
				t.Errorf("got %d rules, want at least %d", len(rules), tt.wantMin)
			}
		})
	}
}

func TestRulesHaveValidFields(t *testing.T) {
	reg := NewRegistry()

	seen := make(map[string]bool)
	for _, rule := range reg.rules {
		// Check for duplicate IDs within a framework
		key := string(rule.Framework) + ":" + rule.ID
		if seen[key] {
			t.Errorf("duplicate rule ID: %s", key)
		}
		seen[key] = true

		if rule.ID == "" {
			t.Error("rule has empty ID")
		}
		if rule.Framework == "" {
			t.Error("rule has empty Framework")
		}
		if rule.Title == "" {
			t.Errorf("rule %s has empty Title", rule.ID)
		}
		if rule.Section == "" {
			t.Errorf("rule %s has empty Section", rule.ID)
		}
		if !rule.Provider.IsValid() {
			t.Errorf("rule %s has invalid provider: %s", rule.ID, rule.Provider)
		}
		if len(rule.ResourceKinds) == 0 {
			t.Errorf("rule %s has no resource kinds", rule.ID)
		}
		for _, rk := range rule.ResourceKinds {
			if !rk.IsValid() {
				t.Errorf("rule %s has invalid resource kind: %s", rule.ID, rk)
			}
		}
	}
}

func TestFormatRulesForPrompt(t *testing.T) {
	reg := NewRegistry()

	rules := reg.GetRulesForResource(
		[]string{"cis_gcp_v4"},
		domain.ProviderGCP,
		domain.ResourceDatabase,
		"Cloud SQL",
	)

	text := reg.FormatRulesForPrompt(rules)

	if text == "" {
		t.Fatal("expected non-empty prompt text")
	}

	// Check that rule IDs appear
	if !containsSubstring(text, "6.4") {
		t.Error("expected rule 6.4 in prompt text")
	}
	if !containsSubstring(text, "require_ssl") {
		t.Error("expected require_ssl attribute in prompt text")
	}

	// Empty rules produces empty output
	empty := reg.FormatRulesForPrompt(nil)
	if empty != "" {
		t.Errorf("expected empty string for nil rules, got %q", empty)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
