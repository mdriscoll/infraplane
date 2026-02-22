package compliance

import "github.com/matthewdriscoll/infraplane/internal/domain"

// registerCISGCP populates the registry with CIS GCP Foundation Benchmark v4.0.0 rules.
// Source: CIS Google Cloud Platform Foundation Benchmark v4.0.0 (05-02-2025)
func (r *Registry) registerCISGCP() {
	r.frameworks[FrameworkCISGCP] = FrameworkInfo{
		ID:          FrameworkCISGCP,
		Name:        "CIS Google Cloud Platform Foundation Benchmark",
		Version:     "4.0.0",
		Provider:    domain.ProviderGCP,
		Description: "Security configuration best practices for Google Cloud Platform",
	}

	r.rules = append(r.rules, cisGCPIAMRules()...)
	r.rules = append(r.rules, cisGCPLoggingRules()...)
	r.rules = append(r.rules, cisGCPNetworkingRules()...)
	r.rules = append(r.rules, cisGCPComputeRules()...)
	r.rules = append(r.rules, cisGCPStorageRules()...)
	r.rules = append(r.rules, cisGCPDatabaseRules()...)
}

// --- Section 1: Identity and Access Management ---

func cisGCPIAMRules() []Rule {
	return []Rule{
		{
			ID:            "1.4",
			Framework:     FrameworkCISGCP,
			Section:       "Identity and Access Management",
			Title:         "Ensure that there are only GCP-managed service account keys for each service account",
			Description:   "User-managed keys introduce risk if not properly rotated. Use GCP-managed keys where possible.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourcePolicy},
			ServiceNames:  []string{"IAM"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_service_account",
					Attribute:    "(avoid creating google_service_account_key resources)",
					Value:        "omit",
					Description:  "Do not create user-managed service account keys; use GCP-managed keys",
				},
			},
		},
		{
			ID:            "1.5",
			Framework:     FrameworkCISGCP,
			Section:       "Identity and Access Management",
			Title:         "Ensure that service account has no admin privileges",
			Description:   "Service accounts should follow least privilege and not be granted admin roles like Owner, Editor, or roles/*Admin.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourcePolicy},
			ServiceNames:  []string{"IAM"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_project_iam_member",
					Attribute:    "role",
					Value:        "must not be roles/owner, roles/editor, or any *Admin role",
					Description:  "Service accounts must not have admin privileges",
				},
			},
		},
		{
			ID:            "1.9",
			Framework:     FrameworkCISGCP,
			Section:       "Identity and Access Management",
			Title:         "Ensure that Cloud KMS cryptokeys are not anonymously or publicly accessible",
			Description:   "KMS keys must not grant allUsers or allAuthenticatedUsers access.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceSecrets},
			ServiceNames:  []string{"Cloud KMS"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_kms_crypto_key_iam_binding",
					Attribute:    "members",
					Value:        "must not include allUsers or allAuthenticatedUsers",
					Description:  "KMS keys must not be publicly accessible",
				},
			},
		},
		{
			ID:            "1.10",
			Framework:     FrameworkCISGCP,
			Section:       "Identity and Access Management",
			Title:         "Ensure KMS encryption keys are rotated within a period of 90 days",
			Description:   "Crypto keys should have automatic rotation configured with a period of 90 days or less.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceSecrets},
			ServiceNames:  []string{"Cloud KMS"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_kms_crypto_key",
					Attribute:    "rotation_period",
					Value:        "7776000s (90 days or less)",
					Description:  "Set automatic key rotation to 90 days or fewer",
				},
			},
		},
	}
}

// --- Section 2: Logging and Monitoring ---

func cisGCPLoggingRules() []Rule {
	return []Rule{
		{
			ID:            "2.1",
			Framework:     FrameworkCISGCP,
			Section:       "Logging and Monitoring",
			Title:         "Ensure that Cloud Audit Logging is configured properly",
			Description:   "Enable Data Access audit logs for all services to track who accessed what data and when.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceCompute, domain.ResourceDatabase, domain.ResourceStorage},
			ServiceNames:  []string{},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_project_iam_audit_config",
					Attribute:    "audit_log_config.log_type",
					Value:        "DATA_READ, DATA_WRITE, ADMIN_READ",
					Description:  "Enable all audit log types for the project",
				},
			},
		},
		{
			ID:            "2.2",
			Framework:     FrameworkCISGCP,
			Section:       "Logging and Monitoring",
			Title:         "Ensure that sinks are configured for all log entries",
			Description:   "Configure a log sink that captures all log entries to a destination (Cloud Storage, BigQuery, or Pub/Sub).",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceCompute, domain.ResourceDatabase, domain.ResourceStorage},
			ServiceNames:  []string{},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_logging_project_sink",
					Attribute:    "filter",
					Value:        "(empty — capture all logs)",
					Description:  "Create a log sink with no filter to capture all entries",
				},
			},
		},
		{
			ID:            "2.12",
			Framework:     FrameworkCISGCP,
			Section:       "Logging and Monitoring",
			Title:         "Ensure that Cloud DNS logging is enabled for all VPC networks",
			Description:   "DNS logging should be enabled on VPC networks to capture DNS queries for security monitoring.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceNetwork},
			ServiceNames:  []string{"VPC"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_dns_policy",
					Attribute:    "enable_logging",
					Value:        "true",
					Description:  "Enable DNS logging on the VPC network",
				},
			},
		},
	}
}

// --- Section 3: Networking ---

func cisGCPNetworkingRules() []Rule {
	return []Rule{
		{
			ID:            "3.1",
			Framework:     FrameworkCISGCP,
			Section:       "Networking",
			Title:         "Ensure that the default network does not exist in a project",
			Description:   "Delete the default VPC network and create custom VPC networks with appropriate subnets and firewall rules.",
			ProfileLevel:  ProfileLevel2,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceNetwork},
			ServiceNames:  []string{"VPC"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_network",
					Attribute:    "auto_create_subnetworks",
					Value:        "false",
					Description:  "Create custom-mode VPC networks, not auto-mode (which replicates the default network)",
				},
			},
		},
		{
			ID:            "3.6",
			Framework:     FrameworkCISGCP,
			Section:       "Networking",
			Title:         "Ensure that SSH access is restricted from the internet",
			Description:   "Firewall rules should not allow SSH (port 22) access from 0.0.0.0/0.",
			ProfileLevel:  ProfileLevel2,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceNetwork, domain.ResourceCompute},
			ServiceNames:  []string{"VPC", "Compute Engine"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_firewall",
					Attribute:    "source_ranges (for port 22 rules)",
					Value:        "must not include 0.0.0.0/0",
					Description:  "Restrict SSH access to specific IP ranges, not the entire internet",
				},
			},
		},
		{
			ID:            "3.7",
			Framework:     FrameworkCISGCP,
			Section:       "Networking",
			Title:         "Ensure that RDP access is restricted from the internet",
			Description:   "Firewall rules should not allow RDP (port 3389) access from 0.0.0.0/0.",
			ProfileLevel:  ProfileLevel2,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceNetwork, domain.ResourceCompute},
			ServiceNames:  []string{"VPC", "Compute Engine"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_firewall",
					Attribute:    "source_ranges (for port 3389 rules)",
					Value:        "must not include 0.0.0.0/0",
					Description:  "Restrict RDP access to specific IP ranges, not the entire internet",
				},
			},
		},
		{
			ID:            "3.8",
			Framework:     FrameworkCISGCP,
			Section:       "Networking",
			Title:         "Ensure that VPC Flow Logs is enabled for every subnet in a VPC network",
			Description:   "VPC Flow Logs capture network flows for monitoring, forensics, and security analysis.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceNetwork},
			ServiceNames:  []string{"VPC"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_subnetwork",
					Attribute:    "log_config.aggregation_interval",
					Value:        "INTERVAL_5_SEC",
					Description:  "Enable VPC flow logs on all subnets",
				},
			},
		},
	}
}

// --- Section 4: Virtual Machines / Compute ---

func cisGCPComputeRules() []Rule {
	return []Rule{
		{
			ID:            "4.1",
			Framework:     FrameworkCISGCP,
			Section:       "Virtual Machines",
			Title:         "Ensure that instances are not configured to use the default service account",
			Description:   "The default Compute Engine service account has the Editor role. Use dedicated service accounts with minimal permissions.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceCompute},
			ServiceNames:  []string{"Compute Engine", "Cloud Run", "GKE"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_instance",
					Attribute:    "service_account.email",
					Value:        "a dedicated service account (not the default compute SA)",
					Description:  "Use a dedicated service account, not the default compute service account",
				},
			},
		},
		{
			ID:            "4.2",
			Framework:     FrameworkCISGCP,
			Section:       "Virtual Machines",
			Title:         "Ensure that instances are not configured to use the default service account with full access to all Cloud APIs",
			Description:   "Limit OAuth scopes to only the APIs required. Never use https://www.googleapis.com/auth/cloud-platform scope with the default SA.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceCompute},
			ServiceNames:  []string{"Compute Engine"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_instance",
					Attribute:    "service_account.scopes",
					Value:        "specific scopes only (not cloud-platform)",
					Description:  "Limit OAuth scopes to required APIs only",
				},
			},
		},
		{
			ID:            "4.4",
			Framework:     FrameworkCISGCP,
			Section:       "Virtual Machines",
			Title:         "Ensure OS Login is enabled for a project",
			Description:   "Enable OS Login to manage SSH access via IAM roles instead of SSH keys.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceCompute},
			ServiceNames:  []string{"Compute Engine"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_project_metadata",
					Attribute:    "metadata.enable-oslogin",
					Value:        "TRUE",
					Description:  "Enable OS Login at the project level",
				},
			},
		},
		{
			ID:            "4.5",
			Framework:     FrameworkCISGCP,
			Section:       "Virtual Machines",
			Title:         "Ensure 'Enable Connecting to Serial Ports' is not enabled for VM instance",
			Description:   "Serial port access should be disabled to prevent interactive console access which bypasses normal authentication.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceCompute},
			ServiceNames:  []string{"Compute Engine"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_instance",
					Attribute:    "metadata.serial-port-enable",
					Value:        "false",
					Description:  "Disable serial port access on VM instances",
				},
			},
		},
		{
			ID:            "4.6",
			Framework:     FrameworkCISGCP,
			Section:       "Virtual Machines",
			Title:         "Ensure that IP forwarding is not enabled on instances",
			Description:   "IP forwarding allows the instance to send/receive packets with non-matching source/destination IPs. Disable unless needed for NAT/routing.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceCompute},
			ServiceNames:  []string{"Compute Engine"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_instance",
					Attribute:    "can_ip_forward",
					Value:        "false",
					Description:  "Disable IP forwarding on VM instances",
				},
			},
		},
		{
			ID:            "4.8",
			Framework:     FrameworkCISGCP,
			Section:       "Virtual Machines",
			Title:         "Ensure Compute instances are launched with Shielded VM enabled",
			Description:   "Shielded VMs provide verifiable integrity via Secure Boot, vTPM, and integrity monitoring.",
			ProfileLevel:  ProfileLevel2,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceCompute},
			ServiceNames:  []string{"Compute Engine"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_instance",
					Attribute:    "shielded_instance_config.enable_secure_boot",
					Value:        "true",
					Description:  "Enable Secure Boot on Shielded VM",
				},
				{
					ResourceType: "google_compute_instance",
					Attribute:    "shielded_instance_config.enable_vtpm",
					Value:        "true",
					Description:  "Enable vTPM on Shielded VM",
				},
				{
					ResourceType: "google_compute_instance",
					Attribute:    "shielded_instance_config.enable_integrity_monitoring",
					Value:        "true",
					Description:  "Enable integrity monitoring on Shielded VM",
				},
			},
		},
		{
			ID:            "4.9",
			Framework:     FrameworkCISGCP,
			Section:       "Virtual Machines",
			Title:         "Ensure that Compute instances do not have public IP addresses",
			Description:   "Instances should use private IPs and access the internet via Cloud NAT or a load balancer.",
			ProfileLevel:  ProfileLevel2,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceCompute},
			ServiceNames:  []string{"Compute Engine"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_instance",
					Attribute:    "network_interface.access_config",
					Value:        "omit (do not assign external IP)",
					Description:  "Do not assign a public IP to compute instances",
				},
			},
		},
		{
			ID:            "4.11",
			Framework:     FrameworkCISGCP,
			Section:       "Virtual Machines",
			Title:         "Ensure that Compute instances have Confidential Computing enabled",
			Description:   "Confidential Computing encrypts data in-use with AMD SEV.",
			ProfileLevel:  ProfileLevel2,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceCompute},
			ServiceNames:  []string{"Compute Engine"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_compute_instance",
					Attribute:    "confidential_instance_config.enable_confidential_compute",
					Value:        "true",
					Description:  "Enable Confidential Computing (AMD SEV) on the instance",
				},
			},
		},
	}
}

// --- Section 5: Storage ---

func cisGCPStorageRules() []Rule {
	return []Rule{
		{
			ID:            "5.1",
			Framework:     FrameworkCISGCP,
			Section:       "Storage",
			Title:         "Ensure that Cloud Storage bucket is not anonymously or publicly accessible",
			Description:   "Storage buckets must not grant access to allUsers or allAuthenticatedUsers.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceStorage},
			ServiceNames:  []string{"Cloud Storage"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_storage_bucket_iam_member",
					Attribute:    "member",
					Value:        "must not be allUsers or allAuthenticatedUsers",
					Description:  "Do not grant public access to storage buckets",
				},
				{
					ResourceType: "google_storage_bucket",
					Attribute:    "public_access_prevention",
					Value:        "enforced",
					Description:  "Enforce public access prevention on the bucket",
				},
			},
		},
		{
			ID:            "5.2",
			Framework:     FrameworkCISGCP,
			Section:       "Storage",
			Title:         "Ensure that Cloud Storage buckets have uniform bucket-level access enabled",
			Description:   "Uniform bucket-level access simplifies permissions by disabling object-level ACLs.",
			ProfileLevel:  ProfileLevel2,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceStorage},
			ServiceNames:  []string{"Cloud Storage"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_storage_bucket",
					Attribute:    "uniform_bucket_level_access",
					Value:        "true",
					Description:  "Enable uniform bucket-level access (disable per-object ACLs)",
				},
			},
		},
	}
}

// --- Section 6: Cloud SQL Database Services ---

func cisGCPDatabaseRules() []Rule {
	return []Rule{
		// Section 6.2: PostgreSQL-specific
		{
			ID:            "6.2.2",
			Framework:     FrameworkCISGCP,
			Section:       "Cloud SQL Database Services",
			Title:         "Ensure that the 'log_connections' database flag for Cloud SQL PostgreSQL instance is set to 'on'",
			Description:   "Enable logging of each attempted connection to the PostgreSQL server.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceDatabase},
			ServiceNames:  []string{"Cloud SQL"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_sql_database_instance",
					Attribute:    "settings.database_flags (name=log_connections)",
					Value:        "on",
					Description:  "Enable connection logging for PostgreSQL",
				},
			},
		},
		{
			ID:            "6.2.3",
			Framework:     FrameworkCISGCP,
			Section:       "Cloud SQL Database Services",
			Title:         "Ensure that the 'log_disconnections' database flag for Cloud SQL PostgreSQL instance is set to 'on'",
			Description:   "Enable logging of session terminations including duration.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceDatabase},
			ServiceNames:  []string{"Cloud SQL"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_sql_database_instance",
					Attribute:    "settings.database_flags (name=log_disconnections)",
					Value:        "on",
					Description:  "Enable disconnection logging for PostgreSQL",
				},
			},
		},
		{
			ID:            "6.2.4",
			Framework:     FrameworkCISGCP,
			Section:       "Cloud SQL Database Services",
			Title:         "Ensure 'log_statement' database flag for Cloud SQL PostgreSQL instance is set appropriately",
			Description:   "Set log_statement to 'ddl' or stricter to log data definition statements.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceDatabase},
			ServiceNames:  []string{"Cloud SQL"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_sql_database_instance",
					Attribute:    "settings.database_flags (name=log_statement)",
					Value:        "ddl",
					Description:  "Log DDL statements at minimum",
				},
			},
		},
		{
			ID:            "6.2.6",
			Framework:     FrameworkCISGCP,
			Section:       "Cloud SQL Database Services",
			Title:         "Ensure 'log_min_error_statement' database flag for Cloud SQL PostgreSQL instance is set to 'error' or stricter",
			Description:   "Set the minimum error severity level that causes a SQL statement to be logged.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceDatabase},
			ServiceNames:  []string{"Cloud SQL"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_sql_database_instance",
					Attribute:    "settings.database_flags (name=log_min_error_statement)",
					Value:        "error",
					Description:  "Log SQL statements that cause errors",
				},
			},
		},
		{
			ID:            "6.2.8",
			Framework:     FrameworkCISGCP,
			Section:       "Cloud SQL Database Services",
			Title:         "Ensure that 'cloudsql.enable_pgaudit' database flag for each Cloud SQL PostgreSQL instance is set to 'on'",
			Description:   "Enable the pgAudit extension for centralized audit logging of PostgreSQL operations.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceDatabase},
			ServiceNames:  []string{"Cloud SQL"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_sql_database_instance",
					Attribute:    "settings.database_flags (name=cloudsql.enable_pgaudit)",
					Value:        "on",
					Description:  "Enable pgAudit extension for centralized audit logging",
				},
			},
		},
		// Section 6.4-6.7: Generic Cloud SQL
		{
			ID:            "6.4",
			Framework:     FrameworkCISGCP,
			Section:       "Cloud SQL Database Services",
			Title:         "Ensure that the Cloud SQL database instance requires all incoming connections to use SSL",
			Description:   "All database connections must use SSL/TLS encryption. Unencrypted connections are rejected.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceDatabase},
			ServiceNames:  []string{"Cloud SQL"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_sql_database_instance",
					Attribute:    "settings.ip_configuration.require_ssl",
					Value:        "true",
					Description:  "Require SSL for all incoming database connections",
				},
			},
		},
		{
			ID:            "6.5",
			Framework:     FrameworkCISGCP,
			Section:       "Cloud SQL Database Services",
			Title:         "Ensure that Cloud SQL database instances do not implicitly whitelist all public IP addresses",
			Description:   "Authorized networks should not include 0.0.0.0/0 which allows any IP to connect.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceDatabase},
			ServiceNames:  []string{"Cloud SQL"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_sql_database_instance",
					Attribute:    "settings.ip_configuration.authorized_networks",
					Value:        "must not include 0.0.0.0/0",
					Description:  "Do not whitelist all public IP addresses in authorized networks",
				},
			},
		},
		{
			ID:            "6.6",
			Framework:     FrameworkCISGCP,
			Section:       "Cloud SQL Database Services",
			Title:         "Ensure that Cloud SQL database instances do not have public IPs",
			Description:   "Database instances should use private IP and be accessed via Private Google Access or Cloud SQL Proxy.",
			ProfileLevel:  ProfileLevel2,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceDatabase},
			ServiceNames:  []string{"Cloud SQL"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_sql_database_instance",
					Attribute:    "settings.ip_configuration.ipv4_enabled",
					Value:        "false",
					Description:  "Disable public IP; use private IP only",
				},
				{
					ResourceType: "google_sql_database_instance",
					Attribute:    "settings.ip_configuration.private_network",
					Value:        "google_compute_network.vpc.id",
					Description:  "Connect the database to a VPC via private networking",
				},
			},
		},
		{
			ID:            "6.7",
			Framework:     FrameworkCISGCP,
			Section:       "Cloud SQL Database Services",
			Title:         "Ensure that Cloud SQL database instances are configured with automated backups",
			Description:   "Automated backups must be enabled to protect against data loss.",
			ProfileLevel:  ProfileLevel1,
			Assessment:    AssessmentAutomated,
			Provider:      domain.ProviderGCP,
			ResourceKinds: []domain.ResourceKind{domain.ResourceDatabase},
			ServiceNames:  []string{"Cloud SQL"},
			TerraformConfigs: []TerraformConfig{
				{
					ResourceType: "google_sql_database_instance",
					Attribute:    "settings.backup_configuration.enabled",
					Value:        "true",
					Description:  "Enable automated backups",
				},
				{
					ResourceType: "google_sql_database_instance",
					Attribute:    "settings.backup_configuration.point_in_time_recovery_enabled",
					Value:        "true",
					Description:  "Enable point-in-time recovery (binary logging for MySQL, WAL for PostgreSQL)",
				},
			},
		},
	}
}
