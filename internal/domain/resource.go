package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ResourceKind represents the type of cloud-agnostic resource.
type ResourceKind string

const (
	ResourceDatabase ResourceKind = "database"
	ResourceCompute  ResourceKind = "compute"
	ResourceStorage  ResourceKind = "storage"
	ResourceCache    ResourceKind = "cache"
	ResourceQueue    ResourceKind = "queue"
	ResourceCDN      ResourceKind = "cdn"
	ResourceNetwork  ResourceKind = "network"
)

// ValidResourceKinds returns all supported resource kinds.
func ValidResourceKinds() []ResourceKind {
	return []ResourceKind{
		ResourceDatabase, ResourceCompute, ResourceStorage,
		ResourceCache, ResourceQueue, ResourceCDN, ResourceNetwork,
	}
}

// IsValid checks whether the resource kind is supported.
func (k ResourceKind) IsValid() bool {
	for _, valid := range ValidResourceKinds() {
		if k == valid {
			return true
		}
	}
	return false
}

// ProviderResource describes how an abstract resource maps to a specific cloud provider.
type ProviderResource struct {
	ServiceName  string         `json:"service_name"`
	Config       map[string]any `json:"config"`
	TerraformHCL string         `json:"terraform_hcl"`
}

// Resource is a cloud-agnostic infrastructure resource belonging to an application.
type Resource struct {
	ID              uuid.UUID                        `json:"id"`
	ApplicationID   uuid.UUID                        `json:"application_id"`
	Kind            ResourceKind                     `json:"kind"`
	Name            string                           `json:"name"`
	Spec            json.RawMessage                  `json:"spec"`
	ProviderMappings map[CloudProvider]ProviderResource `json:"provider_mappings"`
	CreatedAt       time.Time                        `json:"created_at"`
}

// NewResource creates a new resource with the given attributes.
func NewResource(appID uuid.UUID, kind ResourceKind, name string, spec json.RawMessage) Resource {
	return Resource{
		ID:              uuid.New(),
		ApplicationID:   appID,
		Kind:            kind,
		Name:            name,
		Spec:            spec,
		ProviderMappings: make(map[CloudProvider]ProviderResource),
		CreatedAt:       time.Now().UTC(),
	}
}

// Validate checks that the resource has valid required fields.
func (r Resource) Validate() error {
	if r.Name == "" {
		return ErrValidation("resource name is required")
	}
	if !r.Kind.IsValid() {
		return ErrValidation("invalid resource kind: " + string(r.Kind))
	}
	if r.ApplicationID == uuid.Nil {
		return ErrValidation("application ID is required")
	}
	return nil
}
