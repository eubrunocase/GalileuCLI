package guardian

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	sessionID  string
	machineID  string
	auditMutex sync.Mutex
)

type AuditEntry struct {
	Timestamp string `json:"timestamp"`

	RequestID string `json:"request_id"`
	SessionID string `json:"session_id"`
	MachineID string `json:"machine_id"`

	Host     string `json:"host"`
	Provider string `json:"provider"`
	Path     string `json:"path"`
	Method   string `json:"method"`
	Model    string `json:"model,omitempty"`

	Redacted           bool     `json:"redacted"`
	PatternCount       int      `json:"pattern_count"`
	DetectedPatterns   []string `json:"detected_patterns"`
	RedactionPositions []string `json:"redaction_positions"`

	MessageCount    int  `json:"message_count"`
	HasSystemPrompt bool `json:"has_system_prompt"`
	Stream          bool `json:"stream"`

	RequestBodySizeBytes  int `json:"request_body_size_bytes"`
	ResponseBodySizeBytes int `json:"response_body_size_bytes"`
	ResponseStatusCode    int `json:"response_status_code"`

	ProxyLatencyMs     int `json:"proxy_latency_ms"`
	AnalysisDurationMs int `json:"analysis_duration_ms"`

	ProxyError   bool   `json:"proxy_error"`
	ErrorMessage string `json:"error_message,omitempty"`
	WasBlocked   bool   `json:"was_blocked"`
}

type AuditLogger struct {
	mu       sync.Mutex
	file     *os.File
	entries  []AuditEntry
	count    int
	filename string
}

var auditLogger *AuditLogger

func NewAuditLogger(filename string) (*AuditLogger, error) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &AuditLogger{
		file:     f,
		filename: filename,
		entries:  make([]AuditEntry, 0, 100),
		count:    0,
	}, nil
}

func (a *AuditLogger) Log(entry AuditEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.entries = append(a.entries, entry)
	a.count++

	go a.flush()
}

func (a *AuditLogger) flush() {
	if len(a.entries) == 0 {
		return
	}

	for _, e := range a.entries {
		data, _ := json.Marshal(e)
		a.file.Write(append(data, '\n'))
	}

	a.entries = a.entries[:0]
	a.count = 0
}

func (a *AuditLogger) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.flush()
	a.file.Close()
}

func LogAudit(entry AuditEntry) {
	if auditLogger == nil {
		return
	}
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format(time.RFC3339)
	}
	if entry.SessionID == "" {
		entry.SessionID = GetSessionID()
	}
	if entry.MachineID == "" {
		entry.MachineID = GetMachineID()
	}
	auditLogger.Log(entry)
}

func generateSessionID() string {
	return fmt.Sprintf("%x", time.Now().Unix())[:8]
}

func generateMachineID() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	hash := sha256.Sum256([]byte(hostname))
	return hex.EncodeToString(hash[:])[:12]
}

func InitAuditMeta() {
	auditMutex.Lock()
	defer auditMutex.Unlock()

	if sessionID == "" {
		sessionID = generateSessionID()
	}
	if machineID == "" {
		machineID = generateMachineID()
	}
}

func GetSessionID() string {
	auditMutex.Lock()
	defer auditMutex.Unlock()
	return sessionID
}

func GetMachineID() string {
	auditMutex.Lock()
	defer auditMutex.Unlock()
	return machineID
}

func InitAuditLogger() error {
	InitAuditMeta()

	logger, err := NewAuditLogger("galileu_audit.log")
	if err != nil {
		return err
	}
	auditLogger = logger
	fmt.Println("[GALILEU] Logging de auditoria ativo: galileu_audit.log")
	return nil
}

func CloseAuditLogger() {
	if auditLogger != nil {
		auditLogger.Close()
	}
}
