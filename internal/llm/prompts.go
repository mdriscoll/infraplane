package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

const resourceAnalysisSystemPrompt = `You are an expert cloud infrastructure architect. Your job is to analyze natural language descriptions of infrastructure needs and translate them into structured, cloud-agnostic resource definitions.

You must respond with ONLY a JSON object (no markdown, no explanation) with the following structure:

{
  "kind": "database|compute|storage|cache|queue|cdn|network",
  "name": "a-kebab-case-name-for-this-resource",
  "spec": {
    // Kind-specific specification. Examples:
    // database: {"engine": "postgres", "version": "16", "size": "small"}
    // compute: {"runtime": "docker", "cpu": "0.25", "memory": "512MB"}
    // storage: {"type": "object", "access": "private"}
    // cache: {"engine": "redis", "version": "7", "size": "small"}
    // queue: {"type": "standard", "fifo": false}
    // cdn: {"origin_type": "s3"}
    // network: {"type": "vpc", "cidr": "10.0.0.0/16"}
  },
  "mappings": {
    "aws": {
      "service_name": "The AWS service name (e.g. RDS, ElastiCache, S3)",
      "config": {
        // AWS-specific configuration parameters
      },
      "terraform_hcl": "Complete Terraform HCL for this resource on AWS"
    },
    "gcp": {
      "service_name": "The GCP service name (e.g. Cloud SQL, Memorystore, Cloud Storage)",
      "config": {
        // GCP-specific configuration parameters
      },
      "terraform_hcl": "Complete Terraform HCL for this resource on GCP"
    }
  }
}

Guidelines:
- Choose the smallest/cheapest tier appropriate for a development environment
- Generate production-ready Terraform HCL with sensible defaults
- Include both AWS and GCP mappings in every response
- Use descriptive resource names in kebab-case
- The spec should capture the developer's intent in a cloud-agnostic way`

const hostingPlanSystemPrompt = `You are an expert cloud infrastructure architect. Your job is to analyze an application's resources and generate a comprehensive hosting plan.

You must respond with ONLY a JSON object (no markdown, no explanation) with the following structure:

{
  "content": "A detailed hosting plan in Markdown format. Include:\n- Recommended architecture (e.g. ECS Fargate, Cloud Run, etc.)\n- Network topology\n- Security considerations\n- Scaling strategy\n- Monitoring and logging\n- CI/CD pipeline recommendations\n- Complete Terraform configuration for all resources",
  "estimated_cost": {
    "monthly_cost_usd": 150.00,
    "breakdown": {
      "compute": 80.00,
      "database": 50.00,
      "storage": 10.00,
      "networking": 10.00
    }
  }
}

Guidelines:
- Recommend the most cost-effective architecture that meets the application's needs
- Include specific instance types, storage sizes, and configuration
- Estimate costs based on current cloud provider pricing
- Consider high availability, disaster recovery, and security best practices
- The content field should be detailed Markdown with headers, lists, and code blocks`

const migrationPlanSystemPrompt = `You are an expert cloud migration architect. Your job is to generate a detailed migration plan for moving an application from one cloud provider to another.

You must respond with ONLY a JSON object (no markdown, no explanation) with the following structure:

{
  "content": "A detailed migration plan in Markdown format. Include:\n- Service-by-service migration mapping\n- Data migration strategy\n- DNS and networking changes\n- Rollback plan\n- Testing strategy\n- Timeline estimate\n- Risk assessment\n- New Terraform configurations for the target provider",
  "estimated_cost": {
    "monthly_cost_usd": 180.00,
    "breakdown": {
      "compute": 90.00,
      "database": 60.00,
      "storage": 15.00,
      "networking": 15.00
    }
  }
}

Guidelines:
- Map each source service to its closest equivalent on the target provider
- Identify data migration challenges (database dumps, object storage sync, etc.)
- Include a phased migration approach to minimize downtime
- Highlight any feature gaps between providers
- Estimate costs on the target provider
- Include rollback procedures for each phase
- The content field should be detailed Markdown with headers, lists, and code blocks`

func buildResourceAnalysisPrompt(description string, provider domain.CloudProvider) string {
	return fmt.Sprintf(`Analyze the following infrastructure need and generate a cloud-agnostic resource definition with provider mappings for both AWS and GCP.

The user's preferred provider is: %s

User's description:
%s`, provider, description)
}

func buildHostingPlanPrompt(app domain.Application, resources []domain.Resource) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Generate a hosting plan for the following application:\n\n"))
	sb.WriteString(fmt.Sprintf("Application: %s\n", app.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n", app.Description))
	sb.WriteString(fmt.Sprintf("Git Repository: %s\n", app.GitRepoURL))
	sb.WriteString(fmt.Sprintf("Preferred Provider: %s\n\n", app.Provider))

	if len(resources) > 0 {
		sb.WriteString("Resources:\n")
		for _, r := range resources {
			specStr := "{}"
			if len(r.Spec) > 0 {
				specStr = string(r.Spec)
			}
			sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", r.Name, r.Kind, specStr))

			for provider, mapping := range r.ProviderMappings {
				configJSON, _ := json.Marshal(mapping.Config)
				sb.WriteString(fmt.Sprintf("  %s → %s %s\n", provider, mapping.ServiceName, string(configJSON)))
			}
		}
	} else {
		sb.WriteString("No resources defined yet. Recommend a basic hosting setup based on the application description.\n")
	}

	return sb.String()
}

func buildMigrationPlanPrompt(app domain.Application, resources []domain.Resource, from, to domain.CloudProvider) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Generate a migration plan for the following application:\n\n"))
	sb.WriteString(fmt.Sprintf("Application: %s\n", app.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n", app.Description))
	sb.WriteString(fmt.Sprintf("Migrate FROM: %s\n", from))
	sb.WriteString(fmt.Sprintf("Migrate TO: %s\n\n", to))

	if len(resources) > 0 {
		sb.WriteString("Current Resources:\n")
		for _, r := range resources {
			specStr := "{}"
			if len(r.Spec) > 0 {
				specStr = string(r.Spec)
			}
			sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", r.Name, r.Kind, specStr))

			if mapping, ok := r.ProviderMappings[from]; ok {
				configJSON, _ := json.Marshal(mapping.Config)
				sb.WriteString(fmt.Sprintf("  Current (%s): %s %s\n", from, mapping.ServiceName, string(configJSON)))
			}
			if mapping, ok := r.ProviderMappings[to]; ok {
				configJSON, _ := json.Marshal(mapping.Config)
				sb.WriteString(fmt.Sprintf("  Target (%s): %s %s\n", to, mapping.ServiceName, string(configJSON)))
			}
		}
	}

	return sb.String()
}
