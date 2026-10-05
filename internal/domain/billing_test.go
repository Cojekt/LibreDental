package domain

import (
	"regexp"
	"testing"
)

func TestControlNumber_FitsX12Constraints(t *testing.T) {
	pcn := ControlNumber("claim_1759632000000000000")
	if !regexp.MustCompile(`^[A-Z2-7]{16}$`).MatchString(pcn) {
		t.Errorf("control number %q should be 16 uppercase base32 characters", pcn)
	}
	if pcn != ControlNumber("claim_1759632000000000000") || pcn == ControlNumber("claim_1759632000000000001") {
		t.Error("control numbers must be deterministic and distinct per ID")
	}
	c := &Claim{ID: "claim_1759632000000000000"}
	if c.PatientControlNumber() != pcn {
		t.Error("a claim's patient control number must be derived from its ID")
	}
}
