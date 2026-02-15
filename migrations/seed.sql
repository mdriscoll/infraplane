-- Infraplane seed data for local development
-- Run: docker exec infraplane-postgres psql -U infraplane -d infraplane -f /tmp/migrations/seed.sql

-- Clean existing data (in dependency order)
TRUNCATE infrastructure_graphs, infrastructure_plans, deployments, resources, applications CASCADE;

-- ============================================================================
-- Application 1: E-commerce Platform (AWS)
-- ============================================================================
INSERT INTO applications (id, name, description, git_repo_url, source_path, provider, status, created_at, updated_at)
VALUES (
    'a1000000-0000-0000-0000-000000000001',
    'ecommerce-platform',
    'Full-stack e-commerce platform with product catalog, shopping cart, and payment processing',
    'https://github.com/acme/ecommerce-platform',
    '/app',
    'aws',
    'provisioned',
    NOW() - INTERVAL '30 days',
    NOW() - INTERVAL '2 days'
);

-- Resources for ecommerce-platform
INSERT INTO resources (id, application_id, kind, name, spec, provider_mappings, created_at) VALUES
(
    'b1000000-0000-0000-0000-000000000001',
    'a1000000-0000-0000-0000-000000000001',
    'compute',
    'api-server',
    '{"cpu": "2 vCPU", "memory": "4 GB", "runtime": "node:20", "replicas": 3, "port": 8080}',
    '{"aws": {"service_name": "ECS Fargate", "config": {"cluster": "ecommerce-prod", "task_cpu": "1024", "task_memory": "2048"}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '28 days'
),
(
    'b1000000-0000-0000-0000-000000000002',
    'a1000000-0000-0000-0000-000000000001',
    'database',
    'product-db',
    '{"engine": "postgres", "version": "16", "storage_gb": 100, "instance_type": "medium", "multi_az": true}',
    '{"aws": {"service_name": "RDS PostgreSQL", "config": {"instance_class": "db.r6g.large", "storage_type": "gp3"}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '28 days'
),
(
    'b1000000-0000-0000-0000-000000000003',
    'a1000000-0000-0000-0000-000000000001',
    'cache',
    'session-cache',
    '{"engine": "redis", "version": "7", "node_type": "medium", "cluster_mode": false}',
    '{"aws": {"service_name": "ElastiCache Redis", "config": {"node_type": "cache.r6g.large", "num_cache_nodes": 2}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '27 days'
),
(
    'b1000000-0000-0000-0000-000000000004',
    'a1000000-0000-0000-0000-000000000001',
    'cdn',
    'static-assets',
    '{"origins": ["api-server"], "cache_ttl_seconds": 86400, "custom_domain": "cdn.acme-shop.com"}',
    '{"aws": {"service_name": "CloudFront", "config": {"price_class": "PriceClass_100"}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '25 days'
),
(
    'b1000000-0000-0000-0000-000000000005',
    'a1000000-0000-0000-0000-000000000001',
    'storage',
    'product-images',
    '{"type": "object", "versioning": true, "public_read": true}',
    '{"aws": {"service_name": "S3", "config": {"bucket_name": "acme-product-images", "lifecycle_rules": true}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '25 days'
),
(
    'b1000000-0000-0000-0000-000000000006',
    'a1000000-0000-0000-0000-000000000001',
    'queue',
    'order-queue',
    '{"type": "fifo", "retention_days": 14, "max_message_size_kb": 256}',
    '{"aws": {"service_name": "SQS", "config": {"fifo_queue": true, "visibility_timeout": 300}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '24 days'
);

-- Deployments for ecommerce-platform
INSERT INTO deployments (id, application_id, provider, git_commit, git_branch, status, terraform_plan, started_at, completed_at) VALUES
(
    'd1000000-0000-0000-0000-000000000001',
    'a1000000-0000-0000-0000-000000000001',
    'aws',
    'abc123def456789',
    'main',
    'succeeded',
    'Plan: 12 to add, 0 to change, 0 to destroy.',
    NOW() - INTERVAL '20 days',
    NOW() - INTERVAL '20 days' + INTERVAL '5 minutes'
),
(
    'd1000000-0000-0000-0000-000000000002',
    'a1000000-0000-0000-0000-000000000001',
    'aws',
    'def789abc012345',
    'main',
    'succeeded',
    'Plan: 2 to add, 3 to change, 0 to destroy.',
    NOW() - INTERVAL '5 days',
    NOW() - INTERVAL '5 days' + INTERVAL '3 minutes'
);

-- ============================================================================
-- Application 2: Data Pipeline (GCP)
-- ============================================================================
INSERT INTO applications (id, name, description, git_repo_url, source_path, provider, status, created_at, updated_at)
VALUES (
    'a2000000-0000-0000-0000-000000000002',
    'data-pipeline',
    'Real-time data ingestion and processing pipeline for analytics',
    'https://github.com/acme/data-pipeline',
    '/pipeline',
    'gcp',
    'deployed',
    NOW() - INTERVAL '60 days',
    NOW() - INTERVAL '1 day'
);

-- Resources for data-pipeline
INSERT INTO resources (id, application_id, kind, name, spec, provider_mappings, created_at) VALUES
(
    'b2000000-0000-0000-0000-000000000001',
    'a2000000-0000-0000-0000-000000000002',
    'compute',
    'stream-processor',
    '{"cpu": "4 vCPU", "memory": "8 GB", "runtime": "python:3.12", "replicas": 5, "port": 8000}',
    '{"gcp": {"service_name": "Cloud Run", "config": {"min_instances": 2, "max_instances": 10, "cpu_throttling": false}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '55 days'
),
(
    'b2000000-0000-0000-0000-000000000002',
    'a2000000-0000-0000-0000-000000000002',
    'database',
    'analytics-db',
    '{"engine": "postgres", "version": "15", "storage_gb": 500, "instance_type": "large", "high_availability": true}',
    '{"gcp": {"service_name": "Cloud SQL", "config": {"tier": "db-custom-4-16384", "availability_type": "REGIONAL"}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '55 days'
),
(
    'b2000000-0000-0000-0000-000000000003',
    'a2000000-0000-0000-0000-000000000002',
    'queue',
    'event-bus',
    '{"type": "pubsub", "retention_days": 7, "ordering": true}',
    '{"gcp": {"service_name": "Pub/Sub", "config": {"message_retention_duration": "604800s"}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '54 days'
),
(
    'b2000000-0000-0000-0000-000000000004',
    'a2000000-0000-0000-0000-000000000002',
    'storage',
    'raw-data-lake',
    '{"type": "object", "versioning": false, "lifecycle_days": 90}',
    '{"gcp": {"service_name": "Cloud Storage", "config": {"storage_class": "STANDARD", "location": "US"}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '53 days'
),
(
    'b2000000-0000-0000-0000-000000000005',
    'a2000000-0000-0000-0000-000000000002',
    'cache',
    'result-cache',
    '{"engine": "redis", "version": "7", "node_type": "small", "cluster_mode": false}',
    '{"gcp": {"service_name": "Memorystore Redis", "config": {"tier": "STANDARD_HA", "memory_size_gb": 4}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '50 days'
);

-- Deployments for data-pipeline
INSERT INTO deployments (id, application_id, provider, git_commit, git_branch, status, terraform_plan, started_at, completed_at) VALUES
(
    'd2000000-0000-0000-0000-000000000001',
    'a2000000-0000-0000-0000-000000000002',
    'gcp',
    '111aaa222bbb333',
    'main',
    'succeeded',
    'Plan: 8 to add, 0 to change, 0 to destroy.',
    NOW() - INTERVAL '45 days',
    NOW() - INTERVAL '45 days' + INTERVAL '4 minutes'
),
(
    'd2000000-0000-0000-0000-000000000002',
    'a2000000-0000-0000-0000-000000000002',
    'gcp',
    '444ddd555eee666',
    'feature/batch-processing',
    'failed',
    'Error: resource quota exceeded in region us-central1.',
    NOW() - INTERVAL '10 days',
    NOW() - INTERVAL '10 days' + INTERVAL '1 minute'
),
(
    'd2000000-0000-0000-0000-000000000003',
    'a2000000-0000-0000-0000-000000000002',
    'gcp',
    '777ggg888hhh999',
    'main',
    'succeeded',
    'Plan: 1 to add, 2 to change, 0 to destroy.',
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '1 day' + INTERVAL '3 minutes'
);

-- ============================================================================
-- Application 3: Internal Dashboard (AWS) — draft, no resources yet
-- ============================================================================
INSERT INTO applications (id, name, description, git_repo_url, source_path, provider, status, created_at, updated_at)
VALUES (
    'a3000000-0000-0000-0000-000000000003',
    'internal-dashboard',
    'Internal team dashboard for monitoring and metrics visualization',
    'https://github.com/acme/internal-dashboard',
    '/src',
    'aws',
    'draft',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '3 days'
);

-- ============================================================================
-- Application 4: Auth Service (AWS)
-- ============================================================================
INSERT INTO applications (id, name, description, git_repo_url, source_path, provider, status, created_at, updated_at)
VALUES (
    'a4000000-0000-0000-0000-000000000004',
    'auth-service',
    'Centralized authentication and authorization microservice with JWT and OAuth2 support',
    'https://github.com/acme/auth-service',
    '/app',
    'aws',
    'provisioned',
    NOW() - INTERVAL '90 days',
    NOW() - INTERVAL '7 days'
);

-- Resources for auth-service
INSERT INTO resources (id, application_id, kind, name, spec, provider_mappings, created_at) VALUES
(
    'b4000000-0000-0000-0000-000000000001',
    'a4000000-0000-0000-0000-000000000004',
    'compute',
    'auth-api',
    '{"cpu": "1 vCPU", "memory": "2 GB", "runtime": "go:1.22", "replicas": 2, "port": 8443}',
    '{"aws": {"service_name": "ECS Fargate", "config": {"cluster": "shared-services", "task_cpu": "512", "task_memory": "1024"}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '85 days'
),
(
    'b4000000-0000-0000-0000-000000000002',
    'a4000000-0000-0000-0000-000000000004',
    'database',
    'user-db',
    '{"engine": "postgres", "version": "16", "storage_gb": 50, "instance_type": "small", "encrypted": true}',
    '{"aws": {"service_name": "RDS PostgreSQL", "config": {"instance_class": "db.t4g.medium", "storage_encrypted": true}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '85 days'
),
(
    'b4000000-0000-0000-0000-000000000003',
    'a4000000-0000-0000-0000-000000000004',
    'cache',
    'token-cache',
    '{"engine": "redis", "version": "7", "node_type": "small", "cluster_mode": false}',
    '{"aws": {"service_name": "ElastiCache Redis", "config": {"node_type": "cache.t4g.small", "num_cache_nodes": 1}, "terraform_hcl": ""}}',
    NOW() - INTERVAL '84 days'
);

-- Deployment for auth-service
INSERT INTO deployments (id, application_id, provider, git_commit, git_branch, status, terraform_plan, started_at, completed_at) VALUES
(
    'd4000000-0000-0000-0000-000000000001',
    'a4000000-0000-0000-0000-000000000004',
    'aws',
    'aabbccdd11223344',
    'main',
    'succeeded',
    'Plan: 6 to add, 0 to change, 0 to destroy.',
    NOW() - INTERVAL '80 days',
    NOW() - INTERVAL '80 days' + INTERVAL '4 minutes'
);

-- Verify seed data
SELECT 'Applications:' AS info, COUNT(*) AS count FROM applications
UNION ALL
SELECT 'Resources:', COUNT(*) FROM resources
UNION ALL
SELECT 'Deployments:', COUNT(*) FROM deployments;
