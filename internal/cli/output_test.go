package cli

import (
	"strings"
	"sync"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{0, "—"},
		{60000, "1m"},
		{300000, "5m"},
		{3600000, "1h 0m"},
		{5400000, "1h 30m"},
		{7260000, "2h 1m"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.ms)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

func TestPct(t *testing.T) {
	tests := []struct {
		part, whole float64
		want        int
	}{
		{50, 100, 50},
		{1, 3, 33},
		{2, 3, 67},
		{0, 0, 0},
		{0, 100, 0},
		{100, 100, 100},
	}
	for _, tt := range tests {
		got := Pct(tt.part, tt.whole)
		if got != tt.want {
			t.Errorf("Pct(%f, %f) = %d, want %d", tt.part, tt.whole, got, tt.want)
		}
	}
}

func TestProgressBar(t *testing.T) {
	bar := ProgressBar(5, 10, 20)
	if bar == "" {
		t.Error("ProgressBar should not be empty")
	}

	emptyBar := ProgressBar(0, 0, 20)
	if emptyBar == "" {
		t.Error("ProgressBar(0,0) should return empty bar, not empty string")
	}
}

// ---------------------------------------------------------------------------
// Theme Construction
// ---------------------------------------------------------------------------

func TestNewTheme_NotNil(t *testing.T) {
	th := NewTheme()
	if th == nil {
		t.Fatal("NewTheme() returned nil")
	}
}

func TestNewTheme_StylesNotZero(t *testing.T) {
	th := NewTheme()

	// Each semantic style should have a foreground color set (not NoColor{}).
	type namedStyle struct {
		name  string
		style lipgloss.Style
	}
	styles := []namedStyle{
		{"Accent", th.Accent},
		{"Primary", th.Primary},
		{"Secondary", th.Secondary},
		{"Muted", th.Muted},
		{"Success", th.Success},
		{"Error", th.Error},
		{"Warning", th.Warning},
		{"Info", th.Info},
	}

	for _, s := range styles {
		fg := s.style.GetForeground()
		if _, isNoColor := fg.(lipgloss.NoColor); isNoColor {
			t.Errorf("Theme.%s has no foreground color set (zero-value style)", s.name)
		}
	}
}

func TestNewTheme_BadgeStyles(t *testing.T) {
	th := NewTheme()

	type namedStyle struct {
		name  string
		style lipgloss.Style
	}
	badges := []namedStyle{
		{"ErrorBadge", th.ErrorBadge},
		{"SuccessBadge", th.SuccessBadge},
		{"WarnBadge", th.WarnBadge},
		{"InfoBadge", th.InfoBadge},
	}

	for _, b := range badges {
		bg := b.style.GetBackground()
		if _, isNoColor := bg.(lipgloss.NoColor); isNoColor {
			t.Errorf("Theme.%s has no background color set", b.name)
		}
	}
}

func TestGetTheme_Singleton(t *testing.T) {
	a := GetTheme()
	b := GetTheme()
	if a != b {
		t.Error("GetTheme() returned different pointers on successive calls; expected singleton")
	}
}

func TestGetTheme_ConcurrentSafety(t *testing.T) {
	const goroutines = 50
	results := make([]*Theme, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			results[i] = GetTheme()
		}()
	}
	wg.Wait()

	first := results[0]
	for i, th := range results {
		if th != first {
			t.Errorf("goroutine %d: GetTheme() returned a different pointer than goroutine 0", i)
		}
	}
}

// ---------------------------------------------------------------------------
// StatusLine component
// ---------------------------------------------------------------------------

func TestStatusLine_Levels(t *testing.T) {
	th := NewTheme()

	tests := []struct {
		name     string
		level    Level
		wantIcon string
	}{
		{"pass contains checkmark", Pass, "✓"},
		{"fail contains cross", Fail, "✗"},
		{"warn contains warning", Warn, "⚠"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := th.StatusLine(tt.level, "label", "")
			if !strings.Contains(out, tt.wantIcon) {
				t.Errorf("StatusLine(%v) = %q, want it to contain %q", tt.level, out, tt.wantIcon)
			}
		})
	}
}

func TestStatusLine_LabelPresent(t *testing.T) {
	th := NewTheme()
	label := "my-check-label"
	out := th.StatusLine(Pass, label, "")
	if !strings.Contains(out, label) {
		t.Errorf("StatusLine output %q does not contain label %q", out, label)
	}
}

func TestStatusLine_WithDetail(t *testing.T) {
	th := NewTheme()
	detail := "some detail text"
	out := th.StatusLine(Pass, "label", detail)
	if !strings.Contains(out, detail) {
		t.Errorf("StatusLine with detail %q: output %q does not contain detail", detail, out)
	}
}

func TestStatusLine_WithoutDetail_NoTrailingSpaces(t *testing.T) {
	th := NewTheme()
	out := th.StatusLine(Pass, "label", "")
	if strings.HasSuffix(out, "  ") {
		t.Errorf("StatusLine without detail has trailing spaces: %q", out)
	}
}

// ---------------------------------------------------------------------------
// SectionHeader component
// ---------------------------------------------------------------------------

func TestSectionHeader_Uppercase(t *testing.T) {
	th := NewTheme()
	out := th.SectionHeader("diagnostics")
	if !strings.Contains(out, "DIAGNOSTICS") {
		t.Errorf("SectionHeader(%q) = %q, want it to contain %q", "diagnostics", out, "DIAGNOSTICS")
	}
}

func TestSectionHeader_ContainsTitle(t *testing.T) {
	th := NewTheme()
	title := "Results"
	out := th.SectionHeader(title)
	if !strings.Contains(out, strings.ToUpper(title)) {
		t.Errorf("SectionHeader output %q does not contain title %q", out, strings.ToUpper(title))
	}
}

// ---------------------------------------------------------------------------
// Box components (ErrorBox, SuccessBox, WarnBox, InfoBox)
// ---------------------------------------------------------------------------

func TestErrorBox_ContainsMessage(t *testing.T) {
	th := NewTheme()
	msg := "something went wrong"
	out := th.ErrorBox(msg)
	if !strings.Contains(out, msg) {
		t.Errorf("ErrorBox output %q does not contain message %q", out, msg)
	}
}

func TestErrorBox_ContainsDetails(t *testing.T) {
	th := NewTheme()
	detail1 := "first detail line"
	detail2 := "second detail line"
	out := th.ErrorBox("msg", detail1, detail2)
	if !strings.Contains(out, detail1) {
		t.Errorf("ErrorBox output does not contain first detail %q", detail1)
	}
	if !strings.Contains(out, detail2) {
		t.Errorf("ErrorBox output does not contain second detail %q", detail2)
	}
}

func TestErrorBox_NoDetails(t *testing.T) {
	th := NewTheme()
	out := th.ErrorBox("an error occurred")
	if out == "" {
		t.Error("ErrorBox with no details returned empty string")
	}
}

func TestSuccessBox_ContainsMessage(t *testing.T) {
	th := NewTheme()
	msg := "operation completed"
	out := th.SuccessBox(msg)
	if !strings.Contains(out, msg) {
		t.Errorf("SuccessBox output %q does not contain message %q", out, msg)
	}
}

func TestWarnBox_ContainsMessage(t *testing.T) {
	th := NewTheme()
	msg := "proceed with caution"
	out := th.WarnBox(msg)
	if !strings.Contains(out, msg) {
		t.Errorf("WarnBox output %q does not contain message %q", out, msg)
	}
}

func TestInfoBox_ContainsMessage(t *testing.T) {
	th := NewTheme()
	msg := "here is some info"
	out := th.InfoBox(msg)
	if !strings.Contains(out, msg) {
		t.Errorf("InfoBox output %q does not contain message %q", out, msg)
	}
}

// ---------------------------------------------------------------------------
// KeyValue component
// ---------------------------------------------------------------------------

func TestKeyValue_Empty(t *testing.T) {
	th := NewTheme()
	out := th.KeyValue(nil)
	if out != "" {
		t.Errorf("KeyValue(nil) = %q, want empty string", out)
	}
	out = th.KeyValue([]KV{})
	if out != "" {
		t.Errorf("KeyValue([]) = %q, want empty string", out)
	}
}

func TestKeyValue_SinglePair(t *testing.T) {
	th := NewTheme()
	out := th.KeyValue([]KV{{Key: "version", Value: "1.2.3"}})
	if !strings.Contains(out, "version") {
		t.Errorf("KeyValue single pair output %q does not contain key", out)
	}
	if !strings.Contains(out, "1.2.3") {
		t.Errorf("KeyValue single pair output %q does not contain value", out)
	}
}

func TestKeyValue_Alignment(t *testing.T) {
	th := NewTheme()
	pairs := []KV{
		{Key: "short", Value: "v1"},
		{Key: "much-longer-key", Value: "v2"},
		{Key: "mid", Value: "v3"},
	}
	out := th.KeyValue(pairs)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("KeyValue with 3 pairs produced %d lines, want 3", len(lines))
	}

	for _, p := range pairs {
		if !strings.Contains(out, p.Value) {
			t.Errorf("KeyValue output does not contain value %q", p.Value)
		}
	}

	// The value column separator "   " (3 spaces) must appear in every line.
	for i, line := range lines {
		if !strings.Contains(line, "   ") {
			t.Errorf("line %d of KeyValue output %q missing alignment gap", i, line)
		}
	}
}

// ---------------------------------------------------------------------------
// Summary component
// ---------------------------------------------------------------------------

func TestSummary_AllZero(t *testing.T) {
	th := NewTheme()
	out := th.Summary(0, 0, 0)
	if !strings.Contains(out, "no checks run") {
		t.Errorf("Summary(0,0,0) = %q, want it to contain %q", out, "no checks run")
	}
}

func TestSummary_OnlyPass(t *testing.T) {
	th := NewTheme()
	out := th.Summary(5, 0, 0)
	if !strings.Contains(out, "5 passed") {
		t.Errorf("Summary(5,0,0) = %q, want it to contain %q", out, "5 passed")
	}
	if strings.Contains(out, "critical") {
		t.Errorf("Summary(5,0,0) = %q, should not contain 'critical'", out)
	}
	if strings.Contains(out, "warning") {
		t.Errorf("Summary(5,0,0) = %q, should not contain 'warning'", out)
	}
}

func TestSummary_OnlyFail(t *testing.T) {
	th := NewTheme()
	out := th.Summary(0, 3, 0)
	if !strings.Contains(out, "3 critical") {
		t.Errorf("Summary(0,3,0) = %q, want it to contain %q", out, "3 critical")
	}
	if strings.Contains(out, "passed") {
		t.Errorf("Summary(0,3,0) = %q, should not contain 'passed'", out)
	}
}

func TestSummary_Mixed(t *testing.T) {
	th := NewTheme()
	out := th.Summary(4, 2, 1)
	if !strings.Contains(out, "4 passed") {
		t.Errorf("Summary(4,2,1) = %q, want it to contain '4 passed'", out)
	}
	if !strings.Contains(out, "2 critical") {
		t.Errorf("Summary(4,2,1) = %q, want it to contain '2 critical'", out)
	}
	if !strings.Contains(out, "1 warning") {
		t.Errorf("Summary(4,2,1) = %q, want it to contain '1 warning'", out)
	}
}

func TestSummary_WarnPluralization(t *testing.T) {
	th := NewTheme()

	singular := th.Summary(0, 0, 1)
	if !strings.Contains(singular, "1 warning") {
		t.Errorf("Summary(0,0,1) = %q, want singular '1 warning'", singular)
	}

	plural := th.Summary(0, 0, 2)
	if !strings.Contains(plural, "2 warnings") {
		t.Errorf("Summary(0,0,2) = %q, want plural '2 warnings'", plural)
	}
}

// ---------------------------------------------------------------------------
// Banner component
// ---------------------------------------------------------------------------

func TestBanner_WithDetail(t *testing.T) {
	th := NewTheme()
	out := th.Banner("neo", "init ecosystem")
	if !strings.Contains(out, "neo") {
		t.Errorf("Banner output %q does not contain tool name 'neo'", out)
	}
	if !strings.Contains(out, "init ecosystem") {
		t.Errorf("Banner output %q does not contain detail 'init ecosystem'", out)
	}
}

func TestBanner_WithoutDetail(t *testing.T) {
	th := NewTheme()
	out := th.Banner("morpheus", "")
	if !strings.Contains(out, "morpheus") {
		t.Errorf("Banner output %q does not contain tool name 'morpheus'", out)
	}
	if strings.Contains(out, " — ") {
		t.Errorf("Banner without detail %q should not contain ' — ' separator", out)
	}
}

// ---------------------------------------------------------------------------
// Backward Compatibility — package-level functions
// ---------------------------------------------------------------------------

func TestPrintLine_DoesNotPanic(t *testing.T) {
	levels := []struct {
		name  string
		level Level
	}{
		{"Pass", Pass},
		{"Fail", Fail},
		{"Warn", Warn},
	}
	for _, tt := range levels {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PrintLine(%v, ...) panicked: %v", tt.level, r)
				}
			}()
			PrintLine(tt.level, "test label", "some detail")
			PrintLine(tt.level, "test label", "")
		})
	}
}

func TestBold_NotEmpty(t *testing.T) {
	if Bold("test") == "" {
		t.Error("Bold returned empty string")
	}
}

func TestDim_NotEmpty(t *testing.T) {
	if Dim("test") == "" {
		t.Error("Dim returned empty string")
	}
}

func TestGreen_NotEmpty(t *testing.T) {
	if Green("test") == "" {
		t.Error("Green returned empty string")
	}
}

func TestRed_NotEmpty(t *testing.T) {
	if Red("test") == "" {
		t.Error("Red returned empty string")
	}
}

func TestYellow_NotEmpty(t *testing.T) {
	if Yellow("test") == "" {
		t.Error("Yellow returned empty string")
	}
}

func TestCyan_NotEmpty(t *testing.T) {
	if Cyan("test") == "" {
		t.Error("Cyan returned empty string")
	}
}
