package guardian

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
)

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

var bodyPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 32*1024)
	},
}

type LogRequest struct {
	RequestID string
	Host      string
	Provider  string
	Path      string
	Method    string
	Model     string

	Redacted           bool
	PatternCount       int
	DetectedPatterns   []string
	RedactionPositions []string

	MessageCount    int
	HasSystemPrompt bool
	Stream          bool

	RequestBodySizeBytes  int
	ResponseBodySizeBytes int
	ResponseStatusCode    int

	ProxyLatencyMs     int
	AnalysisDurationMs int

	ProxyError   bool
	ErrorMessage string
	WasBlocked   bool
}

type LogWorkerPool struct {
	logChan chan LogRequest
	workers int
	wg      sync.WaitGroup
}

func NewLogWorkerPool(numWorkers int, channelBuffer int) *LogWorkerPool {
	return &LogWorkerPool{
		logChan: make(chan LogRequest, channelBuffer),
		workers: numWorkers,
	}
}

func (p *LogWorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for logReq := range p.logChan {
				entry := AuditEntry{
					RequestID: logReq.RequestID,
					Host:      logReq.Host,
					Provider:  logReq.Provider,
					Path:      logReq.Path,
					Method:    logReq.Method,
					Model:     logReq.Model,

					Redacted:           logReq.Redacted,
					PatternCount:       logReq.PatternCount,
					DetectedPatterns:   logReq.DetectedPatterns,
					RedactionPositions: logReq.RedactionPositions,

					MessageCount:    logReq.MessageCount,
					HasSystemPrompt: logReq.HasSystemPrompt,
					Stream:          logReq.Stream,

					RequestBodySizeBytes:  logReq.RequestBodySizeBytes,
					ResponseBodySizeBytes: logReq.ResponseBodySizeBytes,
					ResponseStatusCode:    logReq.ResponseStatusCode,

					ProxyLatencyMs:     logReq.ProxyLatencyMs,
					AnalysisDurationMs: logReq.AnalysisDurationMs,

					ProxyError:   logReq.ProxyError,
					ErrorMessage: logReq.ErrorMessage,
					WasBlocked:   logReq.WasBlocked,
				}
				LogAudit(entry)
			}
		}()
	}
}

func (p *LogWorkerPool) Submit(logReq LogRequest) {
	select {
	case p.logChan <- logReq:
	default:
		fmt.Printf("[GALILEU] Aviso: fila de logging cheia, descartando log\n")
	}
}

func (p *LogWorkerPool) Shutdown() {
	close(p.logChan)
	p.wg.Wait()
}

var (
	logWorkerPool *LogWorkerPool
	shutdownChan  chan struct{}
)

func GracefulListenWithCA(certPEM, keyPEM []byte) {
	proxy := goproxy.NewProxyHttpServer()

	setCA(certPEM, keyPEM)

	if err := InitAuditLogger(); err != nil {
		fmt.Printf("[GALILEU] Aviso: Nao foi possivel iniciar auditoria: %v\n", err)
	}
	defer CloseAuditLogger()

	logWorkerPool = NewLogWorkerPool(4, 100)
	logWorkerPool.Start()
	defer logWorkerPool.Shutdown()

	analyzer := NewAnalyzer()

	targetHosts := []string{
		"opencode.ai",
		"api.openai.com",
		"api.anthropic.com",
		"generativelanguage.googleapis.com",
		"api.cohere.ai",
		"api.mistral.ai",
	}

	processRequest := func(r *http.Request) (*http.Request, *http.Response) {
		if r.Body == nil {
			return r, nil
		}

		startTime := time.Now()

		bufRaw := bodyPool.Get().([]byte)
		buf := bufRaw[:0]
		defer bodyPool.Put(buf)

		body := bytes.NewBuffer(buf)
		_, err := io.Copy(body, r.Body)
		if err != nil {
			return r, nil
		}
		bodyBytes := body.Bytes()

		analysisStart := time.Now()
		result := analyzer.Analyze(bodyBytes)
		analysisDurationMs := int(time.Since(analysisStart).Milliseconds())

		host, path, method := r.Host, r.URL.Path, r.Method
		provider := inferProvider(host)

		payloadInfo := extractPayloadInfo(bodyBytes)

		logWorkerPool.Submit(LogRequest{
			RequestID: generateUUID(),
			Host:      host,
			Provider:  provider,
			Path:      path,
			Method:    method,
			Model:     payloadInfo.model,

			Redacted:           result.Modified,
			PatternCount:       result.PatternCount,
			DetectedPatterns:   result.DetectedPatterns,
			RedactionPositions: result.RedactionPositions,

			MessageCount:    payloadInfo.messageCount,
			HasSystemPrompt: payloadInfo.hasSystemPrompt,
			Stream:          payloadInfo.stream,

			RequestBodySizeBytes:  len(bodyBytes),
			ResponseBodySizeBytes: 0,

			ProxyLatencyMs:     int(time.Since(startTime).Milliseconds()),
			AnalysisDurationMs: analysisDurationMs,
		})

		if result.Modified {
			fmt.Printf("[GALILEU] Interceptado em %s: Dados sensiveis removidos.\n", r.Host)
			r.Body = io.NopCloser(bytes.NewReader(result.Result))
			r.ContentLength = int64(len(result.Result))
			r.Header.Set("Content-Length", fmt.Sprintf("%d", len(result.Result)))
		} else {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		return r, nil
	}

	for _, host := range targetHosts {
		h := host
		proxy.OnRequest(goproxy.DstHostIs(h)).DoFunc(
			func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
				fmt.Printf("[DEBUG] Requisicao capturada para: %s\n", r.Host)
				return processRequest(r)
			},
		)
	}

	proxy.OnRequest().DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			fmt.Printf("[DEBUG] Requisicao recebida: %s %s\n", r.Method, r.Host)
			return r, nil
		},
	)

	fmt.Println("[GALILEU] Proxy MITM Ativo na porta 9000...")
	shutdownChan = make(chan struct{})

	srv := &http.Server{Addr: ":9000", Handler: proxy}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[GALILEU] Erro no servidor: %v\n", err)
		}
	}()

	<-shutdownChan
}

func GracefulListen() {
	proxy := goproxy.NewProxyHttpServer()

	caCert, err := os.ReadFile("galileu-ca.pem")
	if err != nil {
		fmt.Printf("[GALILEU] Erro ao ler galileu-ca.pem: %v\n", err)
		return
	}
	caKey, err := os.ReadFile("galileu-ca-key.pem")
	if err != nil {
		fmt.Printf("[GALILEU] Erro ao ler galileu-ca-key.pem: %v\n", err)
		return
	}

	setCA(caCert, caKey)

	if err := InitAuditLogger(); err != nil {
		fmt.Printf("[GALILEU] Aviso: Nao foi possivel iniciar auditoria: %v\n", err)
	}
	defer CloseAuditLogger()

	logWorkerPool = NewLogWorkerPool(4, 100)
	logWorkerPool.Start()
	defer logWorkerPool.Shutdown()

	analyzer := NewAnalyzer()

	targetHosts := []string{
		"opencode.ai",
		"api.openai.com",
		"api.anthropic.com",
		"generativelanguage.googleapis.com",
		"api.cohere.ai",
		"api.mistral.ai",
	}

	processRequest := func(r *http.Request) (*http.Request, *http.Response) {
		if r.Body == nil {
			return r, nil
		}

		startTime := time.Now()

		bufRaw := bodyPool.Get().([]byte)
		buf := bufRaw[:0]
		defer bodyPool.Put(buf)

		body := bytes.NewBuffer(buf)
		_, err := io.Copy(body, r.Body)
		if err != nil {
			return r, nil
		}
		bodyBytes := body.Bytes()

		analysisStart := time.Now()
		result := analyzer.Analyze(bodyBytes)
		analysisDurationMs := int(time.Since(analysisStart).Milliseconds())

		host, path, method := r.Host, r.URL.Path, r.Method
		provider := inferProvider(host)

		payloadInfo := extractPayloadInfo(bodyBytes)

		logWorkerPool.Submit(LogRequest{
			RequestID: generateUUID(),
			Host:      host,
			Provider:  provider,
			Path:      path,
			Method:    method,
			Model:     payloadInfo.model,

			Redacted:           result.Modified,
			PatternCount:       result.PatternCount,
			DetectedPatterns:   result.DetectedPatterns,
			RedactionPositions: result.RedactionPositions,

			MessageCount:    payloadInfo.messageCount,
			HasSystemPrompt: payloadInfo.hasSystemPrompt,
			Stream:          payloadInfo.stream,

			RequestBodySizeBytes:  len(bodyBytes),
			ResponseBodySizeBytes: 0,

			ProxyLatencyMs:     int(time.Since(startTime).Milliseconds()),
			AnalysisDurationMs: analysisDurationMs,
		})

		if result.Modified {
			fmt.Printf("[GALILEU] Interceptado em %s: Dados sensiveis removidos.\n", r.Host)
			r.Body = io.NopCloser(bytes.NewReader(result.Result))
			r.ContentLength = int64(len(result.Result))
			r.Header.Set("Content-Length", fmt.Sprintf("%d", len(result.Result)))
		} else {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		return r, nil
	}

	for _, host := range targetHosts {
		h := host
		proxy.OnRequest(goproxy.DstHostIs(h)).DoFunc(
			func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
				fmt.Printf("[DEBUG] Requisicao capturada para: %s\n", r.Host)
				return processRequest(r)
			},
		)
	}

	proxy.OnRequest().DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			fmt.Printf("[DEBUG] Requisicao recebida: %s %s\n", r.Method, r.Host)
			return r, nil
		},
	)

	fmt.Println("[GALILEU] Proxy MITM Ativo na porta 9000...")
	shutdownChan = make(chan struct{})

	srv := &http.Server{Addr: ":9000", Handler: proxy}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[GALILEU] Erro no servidor: %v\n", err)
		}
	}()

	<-shutdownChan
}

func GracefulStop() {
	if shutdownChan != nil {
		close(shutdownChan)
	}
}

func CloseGuardian() {
	GracefulStop()
}

func setCA(caCert, caKey []byte) {
	block, _ := pem.Decode(caCert)
	if block == nil || block.Type != "CERTIFICATE" {
		return
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return
	}

	ca, err := tls.X509KeyPair(caCert, caKey)
	if err != nil {
		return
	}
	ca.Leaf = cert

	goproxy.GoproxyCa = ca
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(&ca)}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(&ca)}
}

func StartGuardian() {
	proxy := goproxy.NewProxyHttpServer()

	caCert, err := os.ReadFile("galileu-ca.pem")
	if err != nil {
		fmt.Printf("[GALILEU] Erro ao ler galileu-ca.pem: %v\n", err)
		return
	}
	caKey, err := os.ReadFile("galileu-ca-key.pem")
	if err != nil {
		fmt.Printf("[GALILEU] Erro ao ler galileu-ca-key.pem: %v\n", err)
		return
	}

	setCA(caCert, caKey)

	if err := InitAuditLogger(); err != nil {
		fmt.Printf("[GALILEU] Aviso: Nao foi possivel iniciar auditoria: %v\n", err)
	}
	defer CloseAuditLogger()

	logWorkerPool = NewLogWorkerPool(4, 100)
	logWorkerPool.Start()
	defer logWorkerPool.Shutdown()

	analyzer := NewAnalyzer()

	targetHosts := []string{
		"opencode.ai",
		"api.openai.com",
		"api.anthropic.com",
		"generativelanguage.googleapis.com",
		"api.cohere.ai",
		"api.mistral.ai",
	}

	processRequest := func(r *http.Request) (*http.Request, *http.Response) {
		if r.Body == nil {
			return r, nil
		}

		startTime := time.Now()

		bufRaw := bodyPool.Get().([]byte)
		buf := bufRaw[:0]
		defer bodyPool.Put(buf)

		body := bytes.NewBuffer(buf)
		_, err := io.Copy(body, r.Body)
		if err != nil {
			return r, nil
		}
		bodyBytes := body.Bytes()

		analysisStart := time.Now()
		result := analyzer.Analyze(bodyBytes)
		analysisDurationMs := int(time.Since(analysisStart).Milliseconds())

		host, path, method := r.Host, r.URL.Path, r.Method
		provider := inferProvider(host)

		payloadInfo := extractPayloadInfo(bodyBytes)

		logWorkerPool.Submit(LogRequest{
			RequestID: generateUUID(),
			Host:      host,
			Provider:  provider,
			Path:      path,
			Method:    method,
			Model:     payloadInfo.model,

			Redacted:           result.Modified,
			PatternCount:       result.PatternCount,
			DetectedPatterns:   result.DetectedPatterns,
			RedactionPositions: result.RedactionPositions,

			MessageCount:    payloadInfo.messageCount,
			HasSystemPrompt: payloadInfo.hasSystemPrompt,
			Stream:          payloadInfo.stream,

			RequestBodySizeBytes:  len(bodyBytes),
			ResponseBodySizeBytes: 0,

			ProxyLatencyMs:     int(time.Since(startTime).Milliseconds()),
			AnalysisDurationMs: analysisDurationMs,
		})

		if result.Modified {
			fmt.Printf("[GALILEU] Interceptado em %s: Dados sensiveis removidos.\n", r.Host)
			r.Body = io.NopCloser(bytes.NewReader(result.Result))
			r.ContentLength = int64(len(result.Result))
			r.Header.Set("Content-Length", fmt.Sprintf("%d", len(result.Result)))
		} else {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		return r, nil
	}

	for _, host := range targetHosts {
		h := host
		proxy.OnRequest(goproxy.DstHostIs(h)).DoFunc(
			func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
				fmt.Printf("[DEBUG] Requisicao capturada para: %s\n", r.Host)
				return processRequest(r)
			},
		)
	}

	proxy.OnRequest().DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			fmt.Printf("[DEBUG] Requisicao recebida: %s %s\n", r.Method, r.Host)
			return r, nil
		},
	)

	fmt.Println("[GALILEU] Proxy MITM Ativo na porta 9000...")
	http.ListenAndServe(":9000", proxy)
}

func inferProvider(host string) string {
	hostLower := strings.ToLower(host)
	if strings.Contains(hostLower, "openai") {
		return "openai"
	}
	if strings.Contains(hostLower, "anthropic") {
		return "anthropic"
	}
	if strings.Contains(hostLower, "google") || strings.Contains(hostLower, "generativelanguage") {
		return "google"
	}
	if strings.Contains(hostLower, "cohere") {
		return "cohere"
	}
	if strings.Contains(hostLower, "mistral") {
		return "mistral"
	}
	return "unknown"
}

type payloadInfo struct {
	model           string
	messageCount    int
	hasSystemPrompt bool
	stream          bool
}

func extractPayloadInfo(body []byte) payloadInfo {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return payloadInfo{}
	}

	info := payloadInfo{}

	if model, ok := data["model"].(string); ok {
		info.model = model
	}

	if stream, ok := data["stream"].(bool); ok {
		info.stream = stream
	}

	if messages, ok := data["messages"].([]interface{}); ok {
		info.messageCount = len(messages)
		for _, m := range messages {
			if msg, ok := m.(map[string]interface{}); ok {
				if role, ok := msg["role"].(string); ok && role == "system" {
					info.hasSystemPrompt = true
					break
				}
			}
		}
	}

	if !info.hasSystemPrompt {
		if _, ok := data["system"]; ok {
			info.hasSystemPrompt = true
		}
	}

	return info
}
