package domain

import "testing"

func TestFormatToothDisplay_FDI(t *testing.T) {
	cases := []struct {
		tooth int
		want  string
	}{
		{1, "18"},   // maxillary right 3rd molar
		{8, "11"},   // maxillary right central incisor
		{9, "21"},   // maxillary left central incisor
		{16, "28"},  // maxillary left 3rd molar
		{17, "38"},  // mandibular left 3rd molar
		{24, "31"},  // mandibular left central incisor
		{25, "41"},  // mandibular right central incisor
		{32, "48"},  // mandibular right 3rd molar
		{101, "55"}, // primary upper right 2nd molar
		{105, "51"}, // primary upper right central incisor
		{111, "75"}, // primary lower left 2nd molar
		{115, "71"}, // primary lower left central incisor
		{116, "81"}, // primary lower right central incisor
		{120, "85"}, // primary lower right 2nd molar
	}
	for _, c := range cases {
		got := FormatToothDisplay(c.tooth, ToothSystemFDI)
		if got != c.want {
			t.Errorf("FormatToothDisplay(%d, FDI) = %q, want %q", c.tooth, got, c.want)
		}
	}
}

func TestFormatToothDisplay_Palmer(t *testing.T) {
	cases := []struct {
		tooth int
		want  string
	}{
		{1, "UR8"},
		{8, "UR1"},
		{9, "UL1"},
		{16, "UL8"},
		{17, "LL8"},
		{24, "LL1"},
		{25, "LR1"},
		{32, "LR8"},
		{111, "LLE"},
		{115, "LLA"},
		{116, "LRA"},
		{120, "LRE"},
	}
	for _, c := range cases {
		got := FormatToothDisplay(c.tooth, ToothSystemPalmer)
		if got != c.want {
			t.Errorf("FormatToothDisplay(%d, Palmer) = %q, want %q", c.tooth, got, c.want)
		}
	}
}
