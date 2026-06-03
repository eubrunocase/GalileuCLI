package tui

import (
	"strings"
	"testing"

	"Galileu/internal/guardian"
)

// --- model.applyEvent ---

func TestApplyEvent_IncrementsTotalOnNonSkipped(t *testing.T) {
	m := New(9000, false)

	m = m.applyEvent(guardian.LogRequest{
		Provider: "openai",
		Host:     "api.openai.com",
	})

	if m.stats.Total != 1 {
		t.Errorf("expected Total=1, got %d", m.stats.Total)
	}
}

func TestApplyEvent_SkippedRequestNotCounted(t *testing.T) {
	m := New(9000, false)

	m = m.applyEvent(guardian.LogRequest{Provider: "skipped"})

	if m.stats.Total != 0 {
		t.Errorf("expected Total=0 for skipped requests, got %d", m.stats.Total)
	}
}

func TestApplyEvent_IncrementsRedactedCount(t *testing.T) {
	m := New(9000, false)

	m = m.applyEvent(guardian.LogRequest{Provider: "anthropic", Redacted: true, PatternCount: 2})
	m = m.applyEvent(guardian.LogRequest{Provider: "anthropic", Redacted: false})

	if m.stats.Redacted != 1 {
		t.Errorf("expected Redacted=1, got %d", m.stats.Redacted)
	}
}

func TestApplyEvent_IncrementsErrorCount(t *testing.T) {
	m := New(9000, false)

	m = m.applyEvent(guardian.LogRequest{Provider: "openai", ProxyError: true})

	if m.stats.Errors != 1 {
		t.Errorf("expected Errors=1, got %d", m.stats.Errors)
	}
}

func TestApplyEvent_TracksProviders(t *testing.T) {
	m := New(9000, false)

	m = m.applyEvent(guardian.LogRequest{Provider: "openai"})
	m = m.applyEvent(guardian.LogRequest{Provider: "openai"})
	m = m.applyEvent(guardian.LogRequest{Provider: "anthropic"})

	if m.stats.Providers["openai"] != 2 {
		t.Errorf("expected openai count=2, got %d", m.stats.Providers["openai"])
	}
	if m.stats.Providers["anthropic"] != 1 {
		t.Errorf("expected anthropic count=1, got %d", m.stats.Providers["anthropic"])
	}
}

func TestApplyEvent_UnknownProviderNotTracked(t *testing.T) {
	m := New(9000, false)

	m = m.applyEvent(guardian.LogRequest{Provider: "unknown"})

	if len(m.stats.Providers) != 0 {
		t.Errorf("expected no providers tracked for 'unknown', got %v", m.stats.Providers)
	}
}

func TestApplyEvent_EmptyProviderNotTracked(t *testing.T) {
	m := New(9000, false)

	m = m.applyEvent(guardian.LogRequest{Provider: ""})

	if len(m.stats.Providers) != 0 {
		t.Errorf("expected no providers tracked for empty provider, got %v", m.stats.Providers)
	}
}

// --- ring buffer ---

func TestApplyEvent_RingBufferCapsAtMaxEvents(t *testing.T) {
	m := New(9000, false)

	for i := 0; i < maxEvents+10; i++ {
		m = m.applyEvent(guardian.LogRequest{Provider: "openai"})
	}

	if len(m.events) != maxEvents {
		t.Errorf("expected events ring to cap at %d, got %d", maxEvents, len(m.events))
	}
}

func TestApplyEvent_SkippedNotAddedToEventList(t *testing.T) {
	m := New(9000, false)

	m = m.applyEvent(guardian.LogRequest{Provider: "skipped"})

	if len(m.events) != 0 {
		t.Errorf("expected 0 events for skipped request, got %d", len(m.events))
	}
}

// --- View (dashboard) ---

func TestView_ContainsPortInHeader(t *testing.T) {
	m := New(9000, false)
	m.width = 120
	m.height = 30

	view := m.View()

	if !strings.Contains(view, ":9000") {
		t.Error("expected view to contain port :9000")
	}
}

func TestView_ContainsDryRunBadgeWhenActive(t *testing.T) {
	m := New(9000, true)
	m.width = 120
	m.height = 30

	view := m.View()

	if !strings.Contains(view, "DRY-RUN") {
		t.Error("expected view to contain DRY-RUN badge when dryRun=true")
	}
}

func TestView_NoDryRunBadgeWhenInactive(t *testing.T) {
	m := New(9000, false)
	m.width = 120
	m.height = 30

	view := m.View()

	if strings.Contains(view, "DRY-RUN") {
		t.Error("expected view to NOT contain DRY-RUN badge when dryRun=false")
	}
}

func TestView_EmptyStateShowsWaitingMessage(t *testing.T) {
	m := New(9000, false)
	m.width = 120
	m.height = 30

	view := m.View()

	if !strings.Contains(view, "Aguardando") {
		t.Error("expected view to show waiting message when no events have arrived")
	}
}

// TestView_RedactedEventTrackedInStats verifies that a redacted event
// increments stats correctly (adapted: badge text differs from original layout).
func TestView_RedactedEventTrackedInStats(t *testing.T) {
	m := New(9000, false)
	m.width = 120
	m.height = 40

	m = m.applyEvent(guardian.LogRequest{
		Provider:         "openai",
		Host:             "api.openai.com",
		Redacted:         true,
		PatternCount:     1,
		DetectedPatterns: []string{"openai_key"},
	})

	if m.stats.Redacted != 1 {
		t.Errorf("expected Redacted stat=1 after redacted event, got %d", m.stats.Redacted)
	}
}

// TestView_CleanEventTrackedInStats verifies a clean (non-redacted) event increments Total.
func TestView_CleanEventTrackedInStats(t *testing.T) {
	m := New(9000, false)
	m.width = 120
	m.height = 40

	m = m.applyEvent(guardian.LogRequest{
		Provider: "anthropic",
		Host:     "api.anthropic.com",
		Redacted: false,
	})

	if m.stats.Total != 1 {
		t.Errorf("expected Total stat=1 after clean event, got %d", m.stats.Total)
	}
	if m.stats.Redacted != 0 {
		t.Errorf("expected Redacted stat=0 for clean event, got %d", m.stats.Redacted)
	}
}

// TestView_ProxyErrorTrackedInStats verifies proxy error increments Errors.
func TestView_ProxyErrorTrackedInStats(t *testing.T) {
	m := New(9000, false)
	m.width = 120
	m.height = 40

	m = m.applyEvent(guardian.LogRequest{
		Provider:   "openai",
		Host:       "api.openai.com",
		ProxyError: true,
	})

	if m.stats.Errors != 1 {
		t.Errorf("expected Errors stat=1 after proxy error event, got %d", m.stats.Errors)
	}
}

// TestView_DashboardContainsGalileuTitle verifies the title appears in the header.
func TestView_DashboardContainsGalileuTitle(t *testing.T) {
	m := New(9000, false)
	m.width = 120
	m.height = 30

	view := m.View()

	if !strings.Contains(view, "GALILEU") {
		t.Error("expected view to contain GALILEU title")
	}
}

// --- New ---

func TestNew_InitialisesProviderMap(t *testing.T) {
	m := New(9000, false)

	if m.stats.Providers == nil {
		t.Error("expected Providers map to be initialised, got nil")
	}
}

func TestNew_StoresPortAndDryRun(t *testing.T) {
	m := New(1234, true)

	if m.port != 1234 {
		t.Errorf("expected port=1234, got %d", m.port)
	}
	if !m.dryRun {
		t.Error("expected dryRun=true")
	}
}

// TestNew_DefaultsToScreenDashboard verifies initial screen is the dashboard.
func TestNew_DefaultsToScreenDashboard(t *testing.T) {
	m := New(9000, false)

	if m.currentScreen != screenDashboard {
		t.Errorf("expected initial screen=screenDashboard, got %d", m.currentScreen)
	}
}

// TestNavigateTo_SwitchesScreen verifies navigateTo changes currentScreen.
func TestNavigateTo_SwitchesScreen(t *testing.T) {
	m := New(9000, false)

	nm, _ := m.navigateTo(screenProxy)

	if nm.currentScreen != screenProxy {
		t.Errorf("expected currentScreen=screenProxy after navigateTo, got %d", nm.currentScreen)
	}
}
