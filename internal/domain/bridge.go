package domain

import "context"

// BridgeCapability describes what context a ProgramBridge can hand to the external program.
type BridgeCapability string

const (
	// BridgeCapabilityPatient means the bridge can open the external program on a patient.
	BridgeCapabilityPatient BridgeCapability = "patient"
	// BridgeCapabilityDocuments means the bridge can open stored documents (e.g. DICOM images)
	// in the external program.
	BridgeCapabilityDocuments BridgeCapability = "documents"
)

// Well-known config keys shared by command-line bridges. Bridge config is per-workstation,
// since the external program is installed (and found at a path) on the local machine.
const (
	BridgeConfigEnabled = "enabled"
	BridgeConfigPath    = "path" // absolute path to the external program's executable
	BridgeConfigArgs    = "args" // argument template, see services.BridgeArgumentTokens
)

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

	// DefaultConfig returns the settings a new workstation starts from.
	DefaultConfig() map[string]string

	// BuildLaunch turns a request and this workstation's config into a launch command.
	// It must not start the program itself.
	BuildLaunch(ctx context.Context, req *BridgeRequest, config map[string]string) (*BridgeLaunch, error)
}

// BridgeInfo describes a registered bridge and its current workstation config for the frontend.
type BridgeInfo struct {
	Name         string             `json:"name"`
	Capabilities []BridgeCapability `json:"capabilities"`
	Config       map[string]string  `json:"config"`
}
