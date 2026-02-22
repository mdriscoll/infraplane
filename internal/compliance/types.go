package compliance

import "github.com/matthewdriscoll/infraplane/internal/domain"

// Framework identifies a compliance framework.
type Framework string

const (
	FrameworkCISGCP Framework = "cis_gcp_v4"
	// Future frameworks:
	// FrameworkCISAWS  Framework = "cis_aws_v3"
	// FrameworkHIPAA   Framework = "hipaa"
	// FrameworkPCIDSS  Framework = "pci_dss_v4"
	// FrameworkSOC2    Framework = "soc2"
)

// ProfileLevel indicates whether a rule is Level 1 (basic) or Level 2 (advanced).
type ProfileLevel int

const (
	ProfileLevel1 ProfileLevel = 1
	ProfileLevel2 ProfileLevel = 2
)

// AssessmentStatus indicates whether a rule can be automated.
type AssessmentStatus string

const (
	AssessmentAutomated AssessmentStatus = "automated"
	AssessmentManual    AssessmentStatus = "manual"
)

// Rule represents a single compliance rule from a benchmark.
type Rule struct {
	ID               string               // e.g., "6.4" or "4.1"
	Framework        Framework            // Which benchmark this belongs to
	Section          string               // e.g., "Virtual Machines", "Cloud SQL Database Services"
	Title            string               // Human-readable title
	Description      string               // What this rule requires
	ProfileLevel     ProfileLevel         // 1 or 2
	Assessment       AssessmentStatus     // automated or manual
	Provider         domain.CloudProvider // gcp, aws
	ResourceKinds    []domain.ResourceKind // Which resource kinds this applies to
	ServiceNames     []string             // Provider-specific service names (e.g., "Cloud SQL", "RDS")
	TerraformConfigs []TerraformConfig    // Specific Terraform settings to enforce
}

// TerraformConfig describes a specific Terraform configuration requirement for compliance.
type TerraformConfig struct {
	ResourceType string // e.g., "google_sql_database_instance"
	Attribute    string // e.g., "settings.ip_configuration.require_ssl"
	Value        string // e.g., "true"
	Description  string // e.g., "Require SSL for all database connections"
}

// FrameworkInfo holds metadata about a compliance framework.
type FrameworkInfo struct {
	ID          Framework            `json:"id"`
	Name        string               `json:"name"`
	Version     string               `json:"version"`
	Provider    domain.CloudProvider `json:"provider"`
	Description string               `json:"description"`
}
