package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/storage"
)

func TestErrorCode(t *testing.T) {
	batchPath := filepath.Join(t.TempDir(), "run.BAT")
	tests := []struct {
		err  error
		want string
	}{
		{ErrUnauthorized, "unauthorized"},
		{fmt.Errorf("failed to load: %w", storage.ErrNotFound), "not_found"},
		{ErrPinNotSet, "incorrect_pin"},
		{ErrPinIncorrect, "incorrect_pin"},
		{validateBridgeExecutable("relative/viewer"), "bridge_path_not_absolute"},
		{validateBridgeExecutable(""), "bridge_path_required"},
		{validateBridgeExecutable(batchPath), "bridge_batch_file"},
		{fmt.Errorf("%w: weasis is not enabled", ErrBridgeNotConfigured), "bridge_not_configured"},
		{validateBridgeArgs("{chart_number}"), "bridge_invalid_args"},
		{errors.New("something else"), ""},
		{nil, ""},
	}
	for _, tt := range tests {
		if got := ErrorCode(tt.err); got != tt.want {
			t.Errorf("ErrorCode(%v) = %q, want %q", tt.err, got, tt.want)
		}
	}
}

func TestErrorCodeKeepsMessage(t *testing.T) {
	// The English text is still what logs and audit details show.
	if got := validateBridgeExecutable("relative"); got.Error() != "program bridge is not configured: program path must be absolute" {
		t.Errorf("unexpected message %q", got)
	}
	if !errors.Is(validateBridgeExecutable("relative"), ErrBridgeNotConfigured) {
		t.Error("path errors must still match ErrBridgeNotConfigured")
	}
}

func TestMarshalError(t *testing.T) {
	if b := MarshalError(errors.New("uncoded")); b != nil {
		t.Errorf("MarshalError for uncoded error = %s, want nil so Wails uses its default", b)
	}
	var cause struct{ Code string }
	if err := json.Unmarshal(MarshalError(fmt.Errorf("wrapped: %w", ErrUnauthorized)), &cause); err != nil {
		t.Fatal(err)
	}
	if cause.Code != "unauthorized" {
		t.Errorf("cause code = %q, want unauthorized", cause.Code)
	}
}
