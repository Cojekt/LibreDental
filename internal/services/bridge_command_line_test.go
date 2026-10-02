package services

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

func TestSplitBridgeArgs(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"-a  -b", []string{"-a", "-b"}},
		{`-d "C:\Program Files\x"`, []string{"-d", `C:\Program Files\x`}},
		{`'$dicom:get -l "{dir}"'`, []string{`$dicom:get -l "{dir}"`}},
		{`--name="{last_name}, {first_name}"`, []string{"--name={last_name}, {first_name}"}},
	}
	for _, tc := range cases {
		got, err := splitBridgeArgs(tc.in)
		if err != nil {
			t.Fatalf("splitBridgeArgs(%q) error: %v", tc.in, err)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("splitBridgeArgs(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	if _, err := splitBridgeArgs(`-d "unterminated`); err == nil {
		t.Error("expected error for unterminated quote")
	}
}

func TestExpandBridgeArgs(t *testing.T) {
	patient := &domain.Patient{
		ID:          "pat_1",
		FirstName:   "Ana Maria",
		LastName:    "O'Neil; rm -rf /",
		DateOfBirth: time.Date(1980, 2, 3, 0, 0, 0, 0, time.UTC),
		Sex:         domain.SexFemale,
	}
	req := &domain.BridgeRequest{
		Patient: patient,
		Dir:     filepath.Join("tmp", "export"),
		Files:   []string{"a.dcm", "b.dcm"},
	}

	got, err := ExpandBridgeArgs(`/id:{patient_id} "{last_name}, {first_name}" {birth_date} {sex} {files} -d "{dir}"`, req)
	if err != nil {
		t.Fatalf("ExpandBridgeArgs error: %v", err)
	}
	want := []string{"/id:pat_1", "O'Neil; rm -rf /, Ana Maria", "19800203", "F", "a.dcm", "b.dcm", "-d", req.Dir}
	if !slices.Equal(got, want) {
		t.Errorf("ExpandBridgeArgs = %q, want %q", got, want)
	}

	errCases := map[string]struct {
		template string
		req      *domain.BridgeRequest
	}{
		"unknown token":          {"{ssn}", req},
		"unclosed token":         {"{last_name", req},
		"files inside arg":       {"--files={files}", req},
		"files without docs":     {"{files}", &domain.BridgeRequest{Patient: patient}},
		"dir without docs":       {"{dir}", &domain.BridgeRequest{Patient: patient}},
		"patient token, no pat":  {"{last_name}", &domain.BridgeRequest{Dir: "x"}},
		"leading dash in value":  {"{last_name}", &domain.BridgeRequest{Patient: &domain.Patient{LastName: "-cl"}}},
		"leading slash in value": {"{last_name}", &domain.BridgeRequest{Patient: &domain.Patient{LastName: "/s"}}},
		"newline in value":       {"{first_name}", &domain.BridgeRequest{Patient: &domain.Patient{FirstName: "a\nb"}}},
		"quote in value":         {"{first_name}", &domain.BridgeRequest{Patient: &domain.Patient{FirstName: `a"b`}}},
		"quote in dir":           {"{dir}", &domain.BridgeRequest{Dir: `a"b`, Files: []string{"f"}}},
	}
	for name, tc := range errCases {
		if _, err := ExpandBridgeArgs(tc.template, tc.req); !errors.Is(err, errBridgeTemplate) {
			t.Errorf("%s: expected template error, got %v", name, err)
		}
	}
}

func TestCommandLineBridgeBuildLaunch(t *testing.T) {
	b := NewCommandLineBridge("weasis", `'$dicom:get -l "{dir}"'`, domain.BridgeCapabilityDocuments)
	exe, _ := filepath.Abs(filepath.Join("bin", "weasis"))
	dir, _ := filepath.Abs("export")

	config := b.DefaultConfig()
	config[domain.BridgeConfigPath] = exe
	launch, err := b.BuildLaunch(context.Background(), &domain.BridgeRequest{Dir: dir, Files: []string{"x"}}, config)
	if err != nil {
		t.Fatalf("BuildLaunch error: %v", err)
	}
	if launch.Executable != exe || !slices.Equal(launch.Args, []string{`$dicom:get -l "` + dir + `"`}) {
		t.Errorf("unexpected launch: %+v", launch)
	}

	for _, path := range []string{"", "relative/weasis", filepath.Join(filepath.Dir(exe), "run.bat"), filepath.Join(filepath.Dir(exe), "run.CMD")} {
		config[domain.BridgeConfigPath] = path
		if _, err := b.BuildLaunch(context.Background(), &domain.BridgeRequest{}, config); !errors.Is(err, ErrBridgeNotConfigured) {
			t.Errorf("path %q: expected ErrBridgeNotConfigured, got %v", path, err)
		}
	}
}

func TestDefaultProgramBridgesTemplatesParse(t *testing.T) {
	seen := map[string]bool{}
	for _, b := range DefaultProgramBridges() {
		if seen[b.Name()] {
			t.Errorf("duplicate bridge name %q", b.Name())
		}
		seen[b.Name()] = true
		if _, err := splitBridgeArgs(b.DefaultConfig()[domain.BridgeConfigArgs]); err != nil {
			t.Errorf("bridge %q default args do not parse: %v", b.Name(), err)
		}
		if b.DefaultConfig()[domain.BridgeConfigEnabled] != "false" {
			t.Errorf("bridge %q should be disabled by default", b.Name())
		}
	}
}
