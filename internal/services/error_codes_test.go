package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/services/bridges"
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
		{bridges.ValidateBridgeExecutable("relative/viewer"), "bridge_path_not_absolute"},
		{bridges.ValidateBridgeExecutable(""), "bridge_path_required"},
		{bridges.ValidateBridgeExecutable(batchPath), "bridge_batch_file"},
		{fmt.Errorf("%w: weasis is not enabled", bridges.ErrBridgeNotConfigured), "bridge_not_configured"},
		{bridges.ValidateBridgeArgs("{chart_number}"), "bridge_invalid_args"},
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
	if got := bridges.ValidateBridgeExecutable("relative"); got.Error() != "program bridge is not configured: program path must be absolute" {
		t.Errorf("unexpected message %q", got)
	}
	if !errors.Is(bridges.ValidateBridgeExecutable("relative"), bridges.ErrBridgeNotConfigured) {
		t.Error("path errors must still match bridges.ErrBridgeNotConfigured")
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
