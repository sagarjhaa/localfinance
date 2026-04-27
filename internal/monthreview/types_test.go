package monthreview

import (
	"testing"
	"time"
)

func TestPeriod_String(t *testing.T) {
	if got := (Period{2026, 4}).String(); got != "2026-04" {
		t.Fatalf("got %q", got)
	}
	if got := (Period{2026, 12}).String(); got != "2026-12" {
		t.Fatalf("got %q", got)
	}
}

func TestParsePeriod(t *testing.T) {
	cases := []struct {
		in      string
		ok      bool
		want    Period
	}{
		{"2026-04", true, Period{2026, 4}},
		{"  2026-12  ", true, Period{2026, 12}},
		{"2026-13", false, Period{}},
		{"2026-00", false, Period{}},
		{"abc-04", false, Period{}},
		{"26-04", false, Period{}},
		{"2026", false, Period{}},
		{"2026-4-1", false, Period{}},
		{"", false, Period{}},
	}
	for _, c := range cases {
		got, err := ParsePeriod(c.in)
		if c.ok && err != nil {
			t.Errorf("ParsePeriod(%q) unexpected error: %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("ParsePeriod(%q) expected error", c.in)
		}
		if c.ok && got != c.want {
			t.Errorf("ParsePeriod(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestPeriod_StartEnd(t *testing.T) {
	p := Period{2026, 4}
	start, end := p.StartEnd()
	wantStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Errorf("start: got %v want %v", start, wantStart)
	}
	if !end.Equal(wantEnd) {
		t.Errorf("end: got %v want %v", end, wantEnd)
	}

	// December rolls into next year.
	p = Period{2026, 12}
	_, end = p.StartEnd()
	if !end.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("dec end: got %v", end)
	}
}
