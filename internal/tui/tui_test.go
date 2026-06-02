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

// --- View ---

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

func TestView_ShowsRedactedBadgeForRedactedRequest(t *testing.T) {
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

	view := m.View()

	if !strings.Contains(view, "REDATADO") {
		t.Error("expected view to show REDATADO badge for redacted request")
	}
}

func TestView_ShowsOKBadgeForCleanRequest(t *testing.T) {
	m := New(9000, false)
	m.width = 120
	m.height = 40

	m = m.applyEvent(guardian.LogRequest{
		Provider: "anthropic",
		Host:     "api.anthropic.com",
		Redacted: false,
	})

	view := m.View()

	if !strings.Contains(view, "OK") {
		t.Error("expected view to show OK badge for clean request")
	}
}

func TestView_ShowsErroBadgeForProxyError(t *testing.T) {
	m := New(9000, false)
	m.width = 120
	m.height = 40

	m = m.applyEvent(guardian.LogRequest{
		Provider:   "openai",
		Host:       "api.openai.com",
		ProxyError: true,
	})

	view := m.View()

	if !strings.Contains(view, "ERRO") {
		t.Error("expected view to show ERRO badge for proxy error")
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
