package services

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/LibreDental/libredental/internal/domain"
)

// CommandLineBridge launches a local program with an argument template, the mechanism most
// dental imaging and charting programs expose (Open Dental calls these "program links").
// Presets for specific programs are just a CommandLineBridge with different default args.
//
// The template is split into arguments before any token is substituted, and the program is
// run without a shell, so patient data can only ever land inside the argument it was placed in.
type CommandLineBridge struct {
	name         string
	capabilities []domain.BridgeCapability
	defaultArgs  string
}

func NewCommandLineBridge(name string, defaultArgs string, capabilities ...domain.BridgeCapability) *CommandLineBridge {
	return &CommandLineBridge{name: name, capabilities: capabilities, defaultArgs: defaultArgs}
}

func (b *CommandLineBridge) Name() string { return b.name }

func (b *CommandLineBridge) Capabilities() []domain.BridgeCapability {
	return append([]domain.BridgeCapability(nil), b.capabilities...)
}

func (b *CommandLineBridge) DefaultConfig() map[string]string {
	return map[string]string{
		domain.BridgeConfigEnabled: "false",
		domain.BridgeConfigPath:    "",
		domain.BridgeConfigArgs:    b.defaultArgs,
	}
}

func (b *CommandLineBridge) BuildLaunch(_ context.Context, req *domain.BridgeRequest, config map[string]string) (*domain.BridgeLaunch, error) {
	exe := config[domain.BridgeConfigPath]
	if err := validateBridgeExecutable(exe); err != nil {
		return nil, err
	}
	args, err := ExpandBridgeArgs(config[domain.BridgeConfigArgs], req)
	if err != nil {
		return nil, err
	}
	return &domain.BridgeLaunch{Executable: exe, Args: args}, nil
}

// DefaultProgramBridges returns the bridges LibreDental ships with. The DICOM viewer presets
// use each viewer's documented command line for opening a local folder of images:
//   - Weasis:     Weasis '$dicom:get -l "<dir>"' (Weasis parses its command string itself)
//   - RadiAnt:    RadiAntViewer.exe -d "<dir>"
//   - MicroDicom: mDicom.exe -fd "<dir>"
func DefaultProgramBridges() []domain.ProgramBridge {
	return []domain.ProgramBridge{
		NewCommandLineBridge("custom", "", domain.BridgeCapabilityPatient, domain.BridgeCapabilityDocuments),
		NewCommandLineBridge("weasis", `'$dicom:get -l "{dir}"'`, domain.BridgeCapabilityDocuments),
		NewCommandLineBridge("radiant", `-d "{dir}"`, domain.BridgeCapabilityDocuments),
		NewCommandLineBridge("microdicom", `-fd "{dir}"`, domain.BridgeCapabilityDocuments),
	}
}

// BridgeArgumentTokens lists the placeholders an argument template may use. {files} must be
// an argument on its own and expands to one argument per exported document.
var BridgeArgumentTokens = []string{
	"{patient_id}", "{first_name}", "{last_name}", "{birth_date}", "{sex}", "{dir}", "{files}",
}

var (
	ErrBridgeNotConfigured = errors.New("program bridge is not configured")
	errBridgeTemplate      = errors.New("invalid bridge argument template")
)

func validateBridgeExecutable(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("%w: program path is required", ErrBridgeNotConfigured)
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%w: program path must be absolute", ErrBridgeNotConfigured)
	}
	// cmd.exe re-parses the command line of batch files with different quoting rules than
	// regular programs, which would undo the argument isolation this bridge relies on.
	switch strings.ToLower(filepath.Ext(path)) {
	case ".bat", ".cmd":
		return fmt.Errorf("%w: batch files cannot be used as bridge programs", ErrBridgeNotConfigured)
	}
	return nil
}

// ExpandBridgeArgs splits an argument template and substitutes tokens from the request.
func ExpandBridgeArgs(template string, req *domain.BridgeRequest) ([]string, error) {
	parts, err := splitBridgeArgs(template)
	if err != nil {
		return nil, err
	}
	if req == nil {
		req = &domain.BridgeRequest{}
	}

	var out []string
	for _, part := range parts {
		if part == "{files}" {
			if len(req.Files) == 0 {
				return nil, fmt.Errorf("%w: {files} requires at least one document", errBridgeTemplate)
			}
			out = append(out, req.Files...)
			continue
		}
		arg, err := expandBridgeArg(part, req)
		if err != nil {
			return nil, err
		}
		out = append(out, arg)
	}
	return out, nil
}

func expandBridgeArg(part string, req *domain.BridgeRequest) (string, error) {
	var sb strings.Builder
	for {
		start := strings.IndexByte(part, '{')
		if start < 0 {
			sb.WriteString(part)
			return sb.String(), nil
		}
		end := strings.IndexByte(part[start:], '}')
		if end < 0 {
			return "", fmt.Errorf("%w: unclosed token in %q", errBridgeTemplate, part)
		}
		token := part[start : start+end+1]
		value, err := bridgeTokenValue(token, req)
		if err != nil {
			return "", err
		}
		sb.WriteString(part[:start])
		sb.WriteString(value)
		part = part[start+end+1:]
	}
}

func bridgeTokenValue(token string, req *domain.BridgeRequest) (string, error) {
	switch token {
	case "{dir}":
		if req.Dir == "" {
			return "", fmt.Errorf("%w: {dir} requires at least one document", errBridgeTemplate)
		}
		if strings.ContainsAny(req.Dir, "\"\n\r") {
			return "", fmt.Errorf("%w: export directory contains unsupported characters", errBridgeTemplate)
		}
		return req.Dir, nil
	case "{files}":
		return "", fmt.Errorf("%w: {files} must be a separate argument", errBridgeTemplate)
	}

	p := req.Patient
	if p == nil {
		return "", fmt.Errorf("%w: %s requires a patient", errBridgeTemplate, token)
	}
	var value string
	switch token {
	case "{patient_id}":
		value = p.ID
	case "{first_name}":
		value = p.FirstName
	case "{last_name}":
		value = p.LastName
	case "{birth_date}":
		// DICOM DA format, which is also what most imaging bridges expect.
		if !p.DateOfBirth.IsZero() {
			value = p.DateOfBirth.Format("20060102")
		}
	case "{sex}":
		value = dicomSex(p.Sex)
	default:
		return "", fmt.Errorf("%w: unknown token %s", errBridgeTemplate, token)
	}
	return sanitizeBridgeValue(token, value)
}

// sanitizeBridgeValue rejects patient values that the receiving program could read as
// something other than data: control characters (line breaks split transfer files and
// some programs' own command parsers), quotes, and a leading dash or slash (option switches).
func sanitizeBridgeValue(token, value string) (string, error) {
	for _, r := range value {
		if unicode.IsControl(r) || r == '"' {
			return "", fmt.Errorf("%w: value for %s contains unsupported characters", errBridgeTemplate, token)
		}
	}
	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("%w: value for %s cannot start with %q", errBridgeTemplate, token, value[:1])
	}
	return value, nil
}

func dicomSex(sex domain.Sex) string {
	switch sex {
	case domain.SexMale:
		return "M"
	case domain.SexFemale:
		return "F"
	case domain.SexOther:
		return "O"
	default:
		return ""
	}
}

// splitBridgeArgs splits on whitespace, honoring single and double quotes for grouping.
// Backslashes are literal so Windows paths can be written as-is. A quote of one kind inside
// a quote of the other is kept, which lets a template pass a pre-quoted command string to
// programs (like Weasis) that parse their own argument.
func splitBridgeArgs(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	var quote rune
	inArg := false
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
			inArg = true
		case unicode.IsSpace(r):
			if inArg {
				args = append(args, cur.String())
				cur.Reset()
				inArg = false
			}
		default:
			cur.WriteRune(r)
			inArg = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("%w: unterminated quote", errBridgeTemplate)
	}
	if inArg {
		args = append(args, cur.String())
	}
	return args, nil
}
