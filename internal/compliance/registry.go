package compliance

import (
	"fmt"
	"strings"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// Registry holds all compliance rules and provides query methods.
type Registry struct {
	rules      []Rule
	frameworks map[Framework]FrameworkInfo
}

// NewRegistry creates a registry pre-loaded with all built-in rules.
func NewRegistry() *Registry {
	r := &Registry{
		frameworks: make(map[Framework]FrameworkInfo),
	}
	r.registerCISGCP()
	// Future: r.registerCISAWS(), r.registerHIPAA(), etc.
	return r
}

// ListFrameworks returns all available framework metadata.
func (r *Registry) ListFrameworks() []FrameworkInfo {
	result := make([]FrameworkInfo, 0, len(r.frameworks))
	for _, info := range r.frameworks {
		result = append(result, info)
	}
	return result
}

// ListFrameworksForProvider returns frameworks applicable to a provider.
func (r *Registry) ListFrameworksForProvider(provider domain.CloudProvider) []FrameworkInfo {
	var result []FrameworkInfo
	for _, info := range r.frameworks {
		if info.Provider == provider {
			result = append(result, info)
		}
	}
	return result
}

// GetFramework returns info for a specific framework.
func (r *Registry) GetFramework(id Framework) (FrameworkInfo, bool) {
	info, ok := r.frameworks[id]
	return info, ok
}

// ValidateFrameworks checks that all framework IDs are valid and match the provider.
func (r *Registry) ValidateFrameworks(ids []string) error {
	for _, id := range ids {
		if _, ok := r.frameworks[Framework(id)]; !ok {
			return fmt.Errorf("unknown compliance framework: %q", id)
		}
	}
	return nil
}

// GetRules returns all rules for the given frameworks.
func (r *Registry) GetRules(frameworks []string) []Rule {
	fwSet := make(map[Framework]bool, len(frameworks))
	for _, id := range frameworks {
		fwSet[Framework(id)] = true
	}

	var result []Rule
	for _, rule := range r.rules {
		if fwSet[rule.Framework] {
			result = append(result, rule)
		}
	}
	return result
}

// GetRulesForResource filters rules to those applicable to a specific
// resource kind, provider, and set of frameworks.
func (r *Registry) GetRulesForResource(
	frameworks []string,
	provider domain.CloudProvider,
	kind domain.ResourceKind,
	serviceName string,
) []Rule {
	fwSet := make(map[Framework]bool, len(frameworks))
	for _, id := range frameworks {
		fwSet[Framework(id)] = true
	}

	serviceNameLower := strings.ToLower(serviceName)

	var result []Rule
	for _, rule := range r.rules {
		if !fwSet[rule.Framework] {
			continue
		}
		if rule.Provider != provider {
			continue
		}

		// Check if this rule applies to the resource kind
		kindMatch := false
		for _, rk := range rule.ResourceKinds {
			if rk == kind {
				kindMatch = true
				break
			}
		}
		if !kindMatch {
			continue
		}

		// If the rule specifies service names, check if our service matches
		if len(rule.ServiceNames) > 0 && serviceName != "" {
			serviceMatch := false
			for _, sn := range rule.ServiceNames {
				if strings.ToLower(sn) == serviceNameLower {
					serviceMatch = true
					break
				}
			}
			if !serviceMatch {
				continue
			}
		}

		result = append(result, rule)
	}
	return result
}

// FormatRulesForPrompt converts a set of rules into a text block
// suitable for injection into LLM prompts.
func (r *Registry) FormatRulesForPrompt(rules []Rule) string {
	if len(rules) == 0 {
		return ""
	}

	var sb strings.Builder

	// Group rules by section for readability
	sections := make(map[string][]Rule)
	var sectionOrder []string
	for _, rule := range rules {
		if _, seen := sections[rule.Section]; !seen {
			sectionOrder = append(sectionOrder, rule.Section)
		}
		sections[rule.Section] = append(sections[rule.Section], rule)
	}

	for _, section := range sectionOrder {
		sectionRules := sections[section]
		sb.WriteString(fmt.Sprintf("### %s\n\n", section))

		for _, rule := range sectionRules {
			sb.WriteString(fmt.Sprintf("**[%s %s]** %s\n", rule.Framework, rule.ID, rule.Title))
			if rule.Description != "" {
				sb.WriteString(fmt.Sprintf("  %s\n", rule.Description))
			}

			if len(rule.TerraformConfigs) > 0 {
				sb.WriteString("  Terraform requirements:\n")
				for _, tc := range rule.TerraformConfigs {
					sb.WriteString(fmt.Sprintf("  - Resource `%s`: set `%s` = `%s`", tc.ResourceType, tc.Attribute, tc.Value))
					if tc.Description != "" {
						sb.WriteString(fmt.Sprintf(" (%s)", tc.Description))
					}
					sb.WriteString("\n")
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
