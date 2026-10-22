package services

import (
	"encoding/json"
	"errors"

	"github.com/LibreDental/libredental/internal/services/bridges"
	"github.com/LibreDental/libredental/internal/storage"
)

// errorCodes maps sentinel errors to stable codes the frontend translates, so staff see a
// localized message instead of Go's English error text (which still goes to logs and audit
// details unchanged). Matched in order with errors.Is, so more specific errors come first.
// Codes must match errorMessages in frontend/src/lib/error.ts.
var errorCodes = []struct {
	err  error
	code string
}{
	{ErrUnauthorized, "unauthorized"},
	{ErrProviderInactive, "provider_inactive"},
	{ErrPinIncorrect, "incorrect_pin"},
	{ErrPinNotSet, "incorrect_pin"},
	{storage.ErrConflict, "conflict"},
	{storage.ErrLastActiveProvider, "last_active_provider"},
	{storage.ErrAlreadyInitialized, "already_initialized"},
	{ErrBridgesUnavailable, "bridges_unavailable"},
	{bridges.ErrBridgePathRequired, "bridge_path_required"},
	{bridges.ErrBridgePathNotAbsolute, "bridge_path_not_absolute"},
	{bridges.ErrBridgeBatchFile, "bridge_batch_file"},
	{bridges.ErrBridgeNotConfigured, "bridge_not_configured"},
	{bridges.ErrBridgeTemplate, "bridge_invalid_args"},
	{storage.ErrNotFound, "not_found"},
}

// ErrorCode returns the frontend error code for err, or "" if it has none.
func ErrorCode(err error) string {
	for _, c := range errorCodes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return ""
}

// MarshalError is Wails' application-level error marshaler: its result becomes the "cause"
// of the rejected call in the frontend. Returning nil keeps Wails' default.
func MarshalError(err error) []byte {
	code := ErrorCode(err)
	if code == "" {
		return nil
	}
	b, _ := json.Marshal(struct {
		Code string `json:"code"`
	}{code})
	return b
}
