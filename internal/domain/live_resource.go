package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// LiveResourceStatus represents the operational state of a live cloud resource.
type LiveResourceStatus string

const (
	LiveResourceActive       LiveResourceStatus = "active"
	LiveResourceProvisioning LiveResourceStatus = "provisioning"
	LiveResourceStopped      LiveResourceStatus = "stopped"
	LiveResourceError        LiveResourceStatus = "error"
	LiveResourceUnknown      LiveResourceStatus = "unknown"
)

// StringMap is a map[string]string that tolerates non-string JSON values
// by converting them to their string representation during unmarshaling.
type StringMap map[string]string

func (m *StringMap) UnmarshalJSON(data []byte) error {
	// Try strict map[string]string first
	var strict map[string]string
	if err := json.Unmarshal(data, &strict); err == nil {
		*m = strict
		return nil
	}

	// Fall back: unmarshal as map[string]any and stringify values
	var loose map[string]any
	if err := json.Unmarshal(data, &loose); err != nil {
		return err
	}
	result := make(map[string]string, len(loose))
	for k, v := range loose {
		result[k] = fmt.Sprintf("%v", v)
	}
	*m = result
	return nil
}

// LiveResource represents an actual deployed cloud resource discovered via CLI or SDK.
type LiveResource struct {
	ResourceType string            `json:"resource_type"` // e.g. "Cloud Run Service", "Cloud SQL Instance"
	Name         string            `json:"name"`
	Provider     CloudProvider     `json:"provider"`
	Region       string            `json:"region"`
	Status       LiveResourceStatus `json:"status"`
	Details      StringMap         `json:"details"`      // provider-specific key-value pairs
	LastChecked  time.Time         `json:"last_checked"`
}

// LiveResourceResult holds the full discovery result for an application.
type LiveResourceResult struct {
	Resources []LiveResource `json:"resources"`
	Errors    []string       `json:"errors,omitempty"` // non-fatal errors (e.g. CLI not found)
	Timestamp time.Time      `json:"timestamp"`
}
