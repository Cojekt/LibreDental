package domain

import (
	"context"
	"time"
)

// BridgeCapability describes what context a ProgramBridge can hand to the external program.
type BridgeCapability string

const (
	// BridgeCapabilityPatient means the bridge can open the external program on a patient.
	BridgeCapabilityPatient BridgeCapability = "patient"
	// BridgeCapabilityDocuments means the bridge can open stored documents (e.g. DICOM images)
	// in the external program.
	BridgeCapabilityDocuments BridgeCapability = "documents"
)

// BridgeConfig is the stored setup for one bridge. Every bridge starts disabled until staff
// point it at the program's executable.
type BridgeConfig struct {
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	Path      string    `json:"path"` // absolute path to the external program's executable
	Args      string    `json:"args"` // argument template, see services.BridgeArgumentTokens
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BridgeRequest is the context a user asked to hand off to an external program.
type BridgeRequest struct {
	Patient *Patient
	// Files are local, absolute paths to exported copies of the requested documents,
	// created by the caller for this launch only.
	Files []string
	// Dir is the directory holding Files; empty when no documents were requested.
	Dir string
}

// BridgeLaunch is how the external program should be started. It is always executed as
// an argv list without a shell, so values substituted into Args can never add arguments
// or commands.
type BridgeLaunch struct {
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
}

// ProgramBridge defines the contract for a local-only integration with another program
// installed on the same workstation (an imaging viewer, a sensor capture app, a perio
// charting tool). Unlike ClaimProvider or NotificationProvider, a bridge never talks to a
// remote service: it launches a local program and passes patient context to it.
type ProgramBridge interface {
	// Name returns the unique identifier for this bridge (e.g., "weasis", "custom").
	Name() string

	// Capabilities returns which kinds of context the bridge accepts.
	Capabilities() []BridgeCapability

	// DefaultConfig returns the settings used until the bridge is configured.
	DefaultConfig() BridgeConfig

	// BuildLaunch turns a request and the bridge's config into a launch command.
	// It must not start the program itself.
	BuildLaunch(ctx context.Context, req *BridgeRequest, config BridgeConfig) (*BridgeLaunch, error)
}

// BridgeInfo describes a registered bridge and its current config for the frontend.
type BridgeInfo struct {
	Capabilities []BridgeCapability `json:"capabilities"`
	Config       BridgeConfig       `json:"config"`
}
