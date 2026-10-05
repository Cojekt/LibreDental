package services

import (
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
)

func TestValidNPI(t *testing.T) {
	for npi, want := range map[string]bool{
		"1234567893":     true, // CMS's published example
		testBillingNPI:   true,
		testRenderingNPI: true,
		"1234567890":     false,
		"123456789":      false,
		"12345678a3":     false,
	} {
		if got := validNPI(npi); got != want {
			t.Errorf("validNPI(%q) = %v, want %v", npi, got, want)
		}
	}
}

func TestX12ToothCode(t *testing.T) {
	for n, want := range map[int]string{1: "1", 32: "32", 101: "A", 103: "C", 120: "T"} {
		if got, ok := x12ToothCode(n); !ok || got != want {
			t.Errorf("x12ToothCode(%d) = %q, %v; want %q", n, got, ok, want)
		}
	}
	for _, n := range []int{0, 33, 100, 121} {
		if _, ok := x12ToothCode(n); ok {
			t.Errorf("x12ToothCode(%d) should be invalid", n)
		}
	}
}

func TestX12ValueFormatting(t *testing.T) {
	if got := x12Phone("+1 (313) 123-4567"); got != "3131234567" {
		t.Errorf("x12Phone = %q", got)
	}
	if got := digitsOnly("12-3456789"); got != "123456789" {
		t.Errorf("digitsOnly should drop separators, got %q", got)
	}
	if got := digitsOnly("12a3456789"); got != "" {
		t.Errorf("digitsOnly should reject stray characters, got %q", got)
	}
	if got := x12Phone("123-4567"); got != "" {
		t.Errorf("x12Phone should reject short numbers, got %q", got)
	}
	if got, err := x12Date("2026-09-28"); err != nil || got != "20260928" {
		t.Errorf("x12Date = %q, %v", got, err)
	}
	if got := centsToDecimal(30550); got != "305.50" {
		t.Errorf("centsToDecimal = %q", got)
	}
	if got := x12Gender(domain.SexUndisclosed); got != "U" {
		t.Errorf("x12Gender(undisclosed) = %q", got)
	}
	if got := x12Text("  Smile*Care ~ Dental  "); got != "Smile Care Dental" {
		t.Errorf("x12Text = %q", got)
	}
}

func TestSplitProviderName(t *testing.T) {
	for name, want := range map[string][2]string{
		"Dr. Jane Q. Smith, DDS": {"Jane Q.", "Smith"},
		"dr Lee":                 {"", "Lee"},
		"Ana Maria Lopez":        {"Ana Maria", "Lopez"},
		"":                       {"", ""},
	} {
		if first, last := splitProviderName(name); first != want[0] || last != want[1] {
			t.Errorf("splitProviderName(%q) = %q, %q; want %q, %q", name, first, last, want[0], want[1])
		}
	}
}
