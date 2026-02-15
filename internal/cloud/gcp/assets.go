// Package gcp provides GCP Cloud Asset Inventory integration for discovering
// live cloud resources within a project.
package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	asset "cloud.google.com/go/asset/apiv1"
	"cloud.google.com/go/asset/apiv1/assetpb"
	"google.golang.org/api/iterator"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// relevantAssetTypes are the GCP asset types we care about for infrastructure discovery.
var relevantAssetTypes = []string{
	"run.googleapis.com/Service",
	"run.googleapis.com/Job",
	"sqladmin.googleapis.com/Instance",
	"storage.googleapis.com/Bucket",
	"secretmanager.googleapis.com/Secret",
	"artifactregistry.googleapis.com/Repository",
	"compute.googleapis.com/Instance",
	"redis.googleapis.com/Instance",
	"pubsub.googleapis.com/Topic",
	"pubsub.googleapis.com/Subscription",
	"cloudfunctions.googleapis.com/Function",
	"vpcaccess.googleapis.com/Connector",
}

// assetTypeToResourceType maps GCP asset types to human-readable resource type names.
var assetTypeToResourceType = map[string]string{
	"run.googleapis.com/Service":                  "Cloud Run Service",
	"run.googleapis.com/Job":                      "Cloud Run Job",
	"sqladmin.googleapis.com/Instance":             "Cloud SQL Instance",
	"storage.googleapis.com/Bucket":                "Cloud Storage Bucket",
	"secretmanager.googleapis.com/Secret":           "Secret Manager Secret",
	"artifactregistry.googleapis.com/Repository":    "Artifact Registry Repository",
	"compute.googleapis.com/Instance":               "Compute Engine Instance",
	"redis.googleapis.com/Instance":                 "Memorystore Redis Instance",
	"pubsub.googleapis.com/Topic":                   "Pub/Sub Topic",
	"pubsub.googleapis.com/Subscription":            "Pub/Sub Subscription",
	"cloudfunctions.googleapis.com/Function":         "Cloud Function",
	"vpcaccess.googleapis.com/Connector":             "VPC Access Connector",
}

// AssetClient wraps the GCP Cloud Asset Inventory API client.
type AssetClient struct {
	client *asset.Client
}

// NewAssetClient creates a new GCP Cloud Asset Inventory client.
// Uses Application Default Credentials (gcloud auth application-default login).
func NewAssetClient(ctx context.Context) (*AssetClient, error) {
	client, err := asset.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create asset client: %w", err)
	}
	return &AssetClient{client: client}, nil
}

// Close releases the underlying gRPC connection.
func (c *AssetClient) Close() error {
	return c.client.Close()
}

// ListProjectAssets discovers all relevant infrastructure resources in a GCP project.
func (c *AssetClient) ListProjectAssets(ctx context.Context, projectID string) ([]domain.LiveResource, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}

	req := &assetpb.ListAssetsRequest{
		Parent:      fmt.Sprintf("projects/%s", projectID),
		AssetTypes:  relevantAssetTypes,
		ContentType: assetpb.ContentType_RESOURCE,
	}

	now := time.Now().UTC()
	var resources []domain.LiveResource

	it := c.client.ListAssets(ctx, req)
	for {
		a, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return resources, fmt.Errorf("list assets: %w", err)
		}

		lr := assetToLiveResource(a, now)
		resources = append(resources, lr)
	}

	return resources, nil
}

// assetToLiveResource converts a GCP asset into a domain LiveResource.
func assetToLiveResource(a *assetpb.Asset, now time.Time) domain.LiveResource {
	name := extractResourceName(a.Name)
	region := extractRegion(a.Name)
	resourceType := assetTypeToResourceType[a.AssetType]
	if resourceType == "" {
		resourceType = a.AssetType
	}

	details := make(map[string]string)
	details["asset_type"] = a.AssetType
	details["full_name"] = a.Name

	// Extract additional details from resource data if available
	if a.Resource != nil && a.Resource.Data != nil {
		fields := a.Resource.Data.GetFields()
		if v, ok := fields["state"]; ok {
			details["state"] = v.GetStringValue()
		}
		if v, ok := fields["status"]; ok {
			if statusFields := v.GetStructValue(); statusFields != nil {
				if cond, ok := statusFields.GetFields()["conditions"]; ok {
					_ = cond // Cloud Run status conditions — complex, just note it
					details["has_conditions"] = "true"
				}
			}
		}
		if v, ok := fields["uri"]; ok {
			details["url"] = v.GetStringValue()
		}
		if v, ok := fields["databaseVersion"]; ok {
			details["database_version"] = v.GetStringValue()
		}
		if v, ok := fields["settings"]; ok {
			if s := v.GetStructValue(); s != nil {
				if tier, ok := s.GetFields()["tier"]; ok {
					details["tier"] = tier.GetStringValue()
				}
			}
		}
	}

	status := inferStatus(a, details)

	return domain.LiveResource{
		ResourceType: resourceType,
		Name:         name,
		Provider:     domain.ProviderGCP,
		Region:       region,
		Status:       status,
		Details:      details,
		LastChecked:  now,
	}
}

// extractResourceName pulls the last segment from a full GCP resource name.
// e.g. "//run.googleapis.com/projects/my-proj/locations/us-central1/services/my-svc" → "my-svc"
func extractResourceName(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

// extractRegion tries to pull a region/location from the full resource name.
func extractRegion(fullName string) string {
	parts := strings.Split(fullName, "/")
	for i, part := range parts {
		if (part == "locations" || part == "regions" || part == "zones") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// inferStatus determines the live status from asset data.
func inferStatus(a *assetpb.Asset, details map[string]string) domain.LiveResourceStatus {
	if state, ok := details["state"]; ok {
		switch strings.ToUpper(state) {
		case "RUNNABLE", "ACTIVE", "READY", "ENABLED":
			return domain.LiveResourceActive
		case "STOPPED", "SUSPENDED", "DISABLED":
			return domain.LiveResourceStopped
		case "PENDING", "CREATING", "PROVISIONING":
			return domain.LiveResourceProvisioning
		case "ERROR", "FAILED":
			return domain.LiveResourceError
		}
	}
	// Most assets that exist in the inventory are active
	return domain.LiveResourceActive
}
