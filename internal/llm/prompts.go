package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/matthewdriscoll/infraplane/internal/analyzer"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

const resourceAnalysisSystemPrompt = `You are an expert cloud infrastructure architect. Your job is to analyze natural language descriptions of infrastructure needs and translate them into structured, cloud-agnostic resource definitions.

You must respond with ONLY a JSON object (no markdown, no explanation) with the following structure:

{
  "kind": "database|compute|storage|cache|queue|cdn|network|secrets|policy",
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
    // secrets: {"secrets": ["DATABASE_URL", "API_KEY"]}
    // policy: {"type": "service_account", "roles": ["roles/cloudsql.client"]}
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

const hostingPlanSystemPrompt = `You are an expert cloud infrastructure architect. Analyze an application's resources and generate a concise hosting plan.

Respond with ONLY a valid JSON object. Do NOT wrap in markdown fences. Do NOT include any text before or after the JSON.

{
  "content": "Hosting plan in Markdown format",
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

CRITICAL: The "content" field must be CONCISE Markdown, strictly under 1500 words. Brevity is essential — be specific but not verbose. Cover:
- Architecture overview: which services to use and why
- Network topology: VPC, subnets, load balancers (brief)
- Security: IAM roles, encryption (brief)
- Scaling strategy (brief)
- Compliance: if compliance frameworks are specified, list which rules each resource satisfies using a compact table or bullet list — do NOT write lengthy explanations per rule

Guidelines:
- Be specific about instance types and configurations
- Do NOT include Terraform code blocks in the content — just mention resource names
- Estimate costs based on current cloud provider pricing
- If compliance requirements are provided, reference rule IDs (e.g. CIS 4.1, CIS 6.4) concisely`

const migrationPlanSystemPrompt = `You are an expert cloud migration architect. Generate a concise migration plan for moving an application between cloud providers.

Respond with ONLY a JSON object (no markdown fences, no explanation):

{
  "content": "Migration plan in Markdown format",
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

The "content" field should be concise Markdown (aim for under 2000 words) covering:
- Service mapping: source → target for each resource
- Data migration strategy per resource type
- Phased timeline with milestones
- Risk assessment and rollback plan
- DNS and networking changes

Guidelines:
- Be specific about target service names and configurations
- Do NOT include full Terraform code — just mention key resource names
- Estimate costs on the target provider
- Include rollback procedures for each phase
- Keep the response focused and actionable`

func buildResourceAnalysisPrompt(description string, provider domain.CloudProvider) string {
	return fmt.Sprintf(`Analyze the following infrastructure need and generate a cloud-agnostic resource definition with provider mappings for both AWS and GCP.

The user's preferred provider is: %s

User's description:
%s`, provider, description)
}

func buildHostingPlanPrompt(app domain.Application, resources []domain.Resource, complianceContext string) string {
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

	if complianceContext != "" {
		sb.WriteString("\n")
		sb.WriteString(complianceContext)
	}

	return sb.String()
}

const codebaseAnalysisSystemPrompt = `You are an expert cloud infrastructure architect. Your job is to analyze application source code files — including dependency manifests, Dockerfiles, configuration files, and deploy scripts — and identify all infrastructure resources the application needs.

You must respond with ONLY a JSON array (no markdown, no explanation) where each element has this structure:

[
  {
    "kind": "database|compute|storage|cache|queue|cdn|network|secrets|policy",
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
      // secrets: {"secrets": ["DATABASE_URL", "API_KEY"]}
      // policy: {"type": "service_account", "roles": ["roles/cloudsql.client"]}
    },
    "mappings": {
      "aws": {
        "service_name": "The AWS service name (e.g. RDS, ElastiCache, S3, ECS)",
        "config": {}
      },
      "gcp": {
        "service_name": "The GCP service name (e.g. Cloud SQL, Memorystore, Cloud Storage, Cloud Run)",
        "config": {}
      }
    }
  }
]

IMPORTANT: Do NOT include terraform_hcl in mappings — Terraform configs are generated separately.

Detection guidelines:
- Look for database drivers/ORMs in dependency files (e.g. pg, prisma, sqlalchemy, gorm) → database resource
- Look for Redis/Memcached clients → cache resource
- Look for message queue clients (SQS, RabbitMQ, Kafka, BullMQ) → queue resource
- Look for S3/storage SDK usage → storage resource
- Look for Dockerfile/docker-compose services → compute resource and any services defined
- Look for environment variables referencing infrastructure (DATABASE_URL, REDIS_URL, etc.)
- Look for existing Terraform files → extract resources defined there
- Look for Kubernetes manifests → identify required resources
- If docker-compose defines services like postgres, redis, etc. → map to managed cloud equivalents
- IMPORTANT: Deploy scripts (deploy/, scripts/) are extremely valuable — they reveal exactly which cloud services the app uses. Look for gcloud, aws, kubectl, terraform commands and extract every cloud resource they create/reference (Cloud Run, Cloud SQL, Artifact Registry, Secret Manager, S3 buckets, etc.)
- CI/CD workflow files (.github/workflows/) also reveal infrastructure dependencies
- Look for Secret Manager, AWS Secrets Manager, SSM Parameter Store, or .env files listing secrets → secrets resource. List all secret names in the spec.
- Look for IAM roles, service accounts, IAM policies, or role bindings → policy resource. List the roles/permissions in the spec.
- Choose the smallest/cheapest tier appropriate for a development environment
- Include both AWS and GCP mappings in every resource
- If no infrastructure resources are detected, return an empty array []`

func buildCodebaseAnalysisPrompt(codeCtx analyzer.CodeContext, provider domain.CloudProvider) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Analyze the following application codebase and identify all infrastructure resources needed.\n\n"))
	sb.WriteString(fmt.Sprintf("Preferred cloud provider: %s\n\n", provider))

	if len(codeCtx.Files) == 0 {
		sb.WriteString("No infrastructure-relevant files were found. Return an empty array [].\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Found %d infrastructure-relevant files:\n\n", len(codeCtx.Files)))
	for _, f := range codeCtx.Files {
		sb.WriteString(fmt.Sprintf("--- %s ---\n", f.Path))
		sb.WriteString(f.Content)
		sb.WriteString("\n\n")
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

const graphSystemPrompt = `You are an expert cloud infrastructure architect. Your job is to analyze an application's resources and generate a topology graph showing how they connect to each other and the public internet.

You must respond with ONLY a JSON object (no markdown, no explanation):

{
  "nodes": [
    {
      "id": "internet",
      "label": "Internet",
      "kind": "internet",
      "service": "Public Internet"
    },
    {
      "id": "unique-node-id",
      "label": "Human-readable name",
      "kind": "compute|database|cache|queue|storage|cdn|network|secrets|policy",
      "service": "The cloud service name (e.g. Cloud Run, RDS, S3)"
    }
  ],
  "edges": [
    {
      "id": "unique-edge-id",
      "source": "source-node-id",
      "target": "target-node-id",
      "label": "Connection description (e.g. HTTPS, TCP/5432, Redis)"
    }
  ]
}

Guidelines:
- ALWAYS include an "internet" node as the entry point for external traffic
- Create a node for every resource provided, using its actual resource ID as the node ID
- The "service" field should use the provider-specific service name (e.g. "Cloud Run" not "compute")
- Edges should flow FROM internet → load balancer/CDN → compute → databases/caches/queues/storage
- Include appropriate protocol labels on edges (HTTPS, TCP/5432, Redis/6379, AMQP, etc.)
- If there's a CDN, internet traffic goes through it first
- If there's a network/VPC, compute resources sit inside it
- Compute resources typically connect to databases, caches, and queues
- Storage can be connected to either compute or CDN
- Secrets resources connect to the compute resources that consume them (e.g. compute → secrets with label "reads secrets")
- Policy resources connect to the compute or service they grant access to (e.g. policy → compute with label "service account", or policy → database with label "IAM binding")
- Every resource must have at least one edge (no orphan nodes except internet)
- Edge IDs should be kebab-case like "internet-to-api" or "api-to-db"`

func buildGraphPrompt(app domain.Application, resources []domain.Resource) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Generate an infrastructure topology graph for the following application:\n\n"))
	sb.WriteString(fmt.Sprintf("Application: %s\n", app.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n", app.Description))
	sb.WriteString(fmt.Sprintf("Provider: %s\n\n", app.Provider))

	if len(resources) > 0 {
		sb.WriteString("Resources:\n")
		for _, r := range resources {
			specStr := "{}"
			if len(r.Spec) > 0 {
				specStr = string(r.Spec)
			}
			sb.WriteString(fmt.Sprintf("- %s (id=%s, kind=%s): %s\n", r.Name, r.ID, r.Kind, specStr))

			for provider, mapping := range r.ProviderMappings {
				configJSON, _ := json.Marshal(mapping.Config)
				sb.WriteString(fmt.Sprintf("  %s → %s %s\n", provider, mapping.ServiceName, string(configJSON)))
			}
		}
	} else {
		sb.WriteString("No resources defined yet. Create a minimal graph with just an internet node.\n")
	}

	return sb.String()
}

const terraformHCLSystemPrompt = `You are an expert cloud infrastructure architect specializing in Terraform. Generate production-ready Terraform HCL for a single cloud resource.

Respond with ONLY a JSON object (no markdown fences, no explanation):

{
  "hcl": "The complete Terraform HCL configuration for this resource"
}

The "hcl" field should contain valid, production-ready Terraform HCL that includes:
- The main resource block with all necessary configuration
- Any required data sources or supporting resources (e.g. IAM roles, security groups)
- Sensible variable declarations for configurable values
- Tags including Name and Environment

Guidelines:
- Use the latest Terraform provider syntax
- Include comments explaining key configuration choices
- Use variables for values that should be configurable
- Follow Terraform best practices for the target provider
- Keep it focused on this single resource — do not include provider blocks
- If compliance requirements are provided, you MUST satisfy every listed rule. Add a comment referencing the rule ID next to each compliance-related attribute (e.g. # CIS 6.4: Require SSL connections)`

func buildTerraformHCLPrompt(resource domain.Resource, provider domain.CloudProvider, complianceContext string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Generate Terraform HCL for the following resource on %s:\n\n", provider))
	sb.WriteString(fmt.Sprintf("Resource Name: %s\n", resource.Name))
	sb.WriteString(fmt.Sprintf("Resource Kind: %s\n", resource.Kind))

	specStr := "{}"
	if len(resource.Spec) > 0 {
		specStr = string(resource.Spec)
	}
	sb.WriteString(fmt.Sprintf("Spec: %s\n", specStr))

	if mapping, ok := resource.ProviderMappings[provider]; ok {
		configJSON, _ := json.Marshal(mapping.Config)
		sb.WriteString(fmt.Sprintf("Provider Service: %s\n", mapping.ServiceName))
		sb.WriteString(fmt.Sprintf("Provider Config: %s\n", string(configJSON)))
	}

	if complianceContext != "" {
		sb.WriteString("\n")
		sb.WriteString(complianceContext)
	}

	return sb.String()
}

const discoveryCommandsSystemPrompt = `You are an expert cloud infrastructure architect. Your job is to analyze deploy scripts and configuration files from an application and generate CLI commands that will list the actual live resources deployed in the cloud.

You must respond with ONLY a JSON object (no markdown, no explanation):

{
  "commands": [
    {
      "description": "Human-readable description of what this checks",
      "command": "The exact CLI command to run",
      "resource_type": "The type of resource this discovers (e.g. Cloud Run Service, Cloud SQL Instance)"
    }
  ]
}

CRITICAL SAFETY RULES:
- ONLY generate read-only commands (list, describe). NEVER generate commands that create, modify, or delete resources.
- For GCP: Only use gcloud subcommands: list, describe. NEVER use deploy, create, delete, update, set-iam-policy.
- For AWS: Only use aws subcommands: list-*, describe-*, get-*. NEVER use create-*, delete-*, put-*, update-*.
- Always include --format=json for gcloud commands or --output json for aws commands.
- Always include --project and --region flags for gcloud commands (extract these from deploy scripts).
- Always include --region flag for aws commands.
- Extract project ID, region, service names, instance names, and other identifiers from the deploy scripts.
- If a secret name is referenced, generate a command to check if the secret exists (gcloud secrets describe), NOT to read its value. NEVER use "secrets versions access".

Guidelines:
- Parse deploy scripts carefully for resource names, project IDs, regions, and service names
- Generate one command per resource type discovered in the scripts
- Include commands for ALL resource types found (Cloud Run, Cloud SQL, Artifact Registry, Secret Manager, S3, RDS, ECS, etc.)
- If the scripts use environment variables for project/region, hardcode the actual values you see in the scripts (look for variable assignments like PROJECT_ID=xxx or defaults)
- If you cannot determine the project or region, use the placeholder values $GOOGLE_PROJECT and us-central1 for GCP, or $AWS_REGION for AWS
- Prefer "describe" for specific named resources (to get detailed status) and "list" for resource types where you want to see all instances`

func buildDiscoveryCommandsPrompt(app domain.Application, codeCtx analyzer.CodeContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Analyze the following deploy scripts and generate CLI commands to discover live cloud resources.\n\n"))
	sb.WriteString(fmt.Sprintf("Application: %s\n", app.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n", app.Description))
	sb.WriteString(fmt.Sprintf("Provider: %s\n\n", app.Provider))

	if len(codeCtx.Files) == 0 {
		sb.WriteString("No deploy scripts or infrastructure files were found. Return an empty commands array.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Found %d infrastructure-relevant files:\n\n", len(codeCtx.Files)))
	for _, f := range codeCtx.Files {
		sb.WriteString(fmt.Sprintf("--- %s ---\n", f.Path))
		sb.WriteString(f.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

const discoveryOutputParseSystemPrompt = `You are an expert cloud infrastructure architect. Parse the following CLI output from cloud provider commands and extract structured information about each live resource.

You must respond with ONLY a JSON object (no markdown, no explanation):

{
  "resources": [
    {
      "resource_type": "Cloud Run Service",
      "name": "family-calendar",
      "provider": "gcp",
      "region": "us-central1",
      "status": "active",
      "details": {
        "key1": "value1",
        "key2": "value2"
      }
    }
  ]
}

Guidelines for the status field:
- "active" if the resource is running, READY, RUNNABLE, ACTIVE, or serving traffic
- "provisioning" if creating or updating
- "stopped" if paused, SUSPENDED, STOPPED, or DISABLED
- "error" if in a failure state
- "unknown" if the status cannot be determined

Guidelines for the details field — include relevant provider-specific metadata:
- Cloud Run: url, memory, cpu, min_instances, max_instances, last_deployed, image
- Cloud SQL: tier, database_version, storage_size_gb, connection_name, ip_address
- Artifact Registry: format, location, repository_size
- Secret Manager: state, version_count, create_time
- S3: region, creation_date, versioning
- RDS: engine, engine_version, instance_class, endpoint
- ECS: launch_type, desired_count, running_count

If a command returned an error or "not found", skip that resource entirely.
If a command output lists multiple resources, create a separate entry for each.
Only include resources that actually exist — do not invent resources.`

func buildDiscoveryOutputParsePrompt(app domain.Application, commandOutputs []CommandOutput) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Parse the following CLI outputs and extract live resource information.\n\n"))
	sb.WriteString(fmt.Sprintf("Application: %s\n", app.Name))
	sb.WriteString(fmt.Sprintf("Provider: %s\n\n", app.Provider))

	for i, co := range commandOutputs {
		sb.WriteString(fmt.Sprintf("--- Command %d: %s ---\n", i+1, co.Command.Description))
		sb.WriteString(fmt.Sprintf("Resource Type: %s\n", co.Command.ResourceType))
		sb.WriteString(fmt.Sprintf("Command: %s\n", co.Command.Command))
		if co.Error != "" {
			sb.WriteString(fmt.Sprintf("Error: %s\n", co.Error))
		}
		if co.Output != "" {
			sb.WriteString(fmt.Sprintf("Output:\n%s\n", co.Output))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
