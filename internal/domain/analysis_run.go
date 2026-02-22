package domain

import (
	"time"

	"github.com/google/uuid"
)

// AnalysisRunTrigger describes what initiated an analysis run.
type AnalysisRunTrigger string

const (
	RunTriggerRegister  AnalysisRunTrigger = "register"
	RunTriggerReanalyze AnalysisRunTrigger = "reanalyze"
	RunTriggerUpload    AnalysisRunTrigger = "upload"
)

// AnalysisRun tracks a single code analysis execution and groups the resources
// that were detected during that run.
type AnalysisRun struct {
	ID            uuid.UUID          `json:"id"`
	ApplicationID uuid.UUID          `json:"application_id"`
	Trigger       AnalysisRunTrigger `json:"trigger"`
	ResourceCount int                `json:"resource_count"`
	CreatedAt     time.Time          `json:"created_at"`
}

// NewAnalysisRun creates a new AnalysisRun with a fresh UUID and timestamp.
func NewAnalysisRun(appID uuid.UUID, trigger AnalysisRunTrigger) AnalysisRun {
	return AnalysisRun{
		ID:            uuid.New(),
		ApplicationID: appID,
		Trigger:       trigger,
		CreatedAt:     time.Now().UTC(),
	}
}
