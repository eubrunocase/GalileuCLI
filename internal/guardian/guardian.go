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

func GracefulListenWithCA(certPEM, keyPEM []byte, analyzer *Analyzer, port int) {
	proxy := goproxy.NewProxyHttpServer()

	setCA(certPEM, keyPEM)

	if err := InitAuditLogger(); err != nil {
		fmt.Printf("[GALILEU] Aviso: Nao foi possivel iniciar auditoria: %v\n", err)
	}

	logWorkerPool = NewLogWorkerPool(4, 100)
	logWorkerPool.Start()
	defer logWorkerPool.Shutdown()

	targetHosts := []string{
		"opencode.ai",
		"api.openai.com",
		"api.anthropic.com",
		"generativelanguage.googleapis.com",
		"aistudio.googleapis.com",
		"api.cohere.ai",
		"api.mistral.ai",
	}

	for _, host := range targetHosts {
		h := host
		proxy.OnRequest(goproxy.DstHostIs(h)).DoFunc(
			func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
				if r.Body == nil || r.Body == http.NoBody || r.Method != "POST" {
					return r, nil
				}

				startTime := time.Now()

				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					logWorkerPool.Submit(LogRequest{
						RequestID:      generateUUID(),
						Host:           r.Host,
						Provider:       inferProvider(r.Host),
						Path:           r.URL.Path,
						Method:         r.Method,
						ProxyError:     true,
						ErrorMessage:   fmt.Sprintf("Failed to read body: %v", err),
						ProxyLatencyMs: int(time.Since(startTime).Milliseconds()),
					})
					return r, nil
				}

				analysisStart := time.Now()
				result := analyzer.Analyze(bodyBytes)
				analysisDurationMs := int(time.Since(analysisStart).Milliseconds())

				host := r.Host
				path := r.URL.Path
				method := r.Method
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
					fmt.Printf("[GALILEU] Interceptado em %s: %d dado(s) sensivel(is) removido(s).\n", r.Host, result.PatternCount)
					r.Body = io.NopCloser(bytes.NewReader(result.Result))
					r.ContentLength = int64(len(result.Result))
					r.Header.Set("Content-Length", fmt.Sprintf("%d", len(result.Result)))
					if r.Header.Get("Transfer-Encoding") != "" {
						r.Header.Del("Transfer-Encoding")
					}
				} else {
					// ✅ CORREÇÃO: Restaurar body sem re-ler
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
					r.ContentLength = int64(len(bodyBytes))
				}

				return r, nil
			},
		)
	}

	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp != nil && resp.StatusCode >= 500 {
			fmt.Printf("[GALILEU] Erro %d recebido de %s%s\n", resp.StatusCode, ctx.Req.Host, ctx.Req.URL.Path)
		}
		return resp
	})

	// ✅ CORREÇÃO: Adicionar handler para erros de conexão usando ConnectionErrHandler
	proxy.ConnectionErrHandler = func(conn io.Writer, ctx *goproxy.ProxyCtx, err error) {
		fmt.Printf("[GALILEU] ❌ ERRO DE CONEXÃO: %v\n[GALILEU] Host: %s | Path: %s | Method: %s\n",
			err, ctx.Req.Host, ctx.Req.URL.Path, ctx.Req.Method)

		logWorkerPool.Submit(LogRequest{
			RequestID:    generateUUID(),
			Host:         ctx.Req.Host,
			Provider:     inferProvider(ctx.Req.Host),
			Path:         ctx.Req.URL.Path,
			Method:       ctx.Req.Method,
			ProxyError:   true,
			ErrorMessage: fmt.Sprintf("Connection error: %v", err),
		})

		errorResponse := fmt.Sprintf(
			"HTTP/1.1 502 Bad Gateway\r\n"+
				"Content-Type: text/plain\r\n"+
				"Content-Length: %d\r\n\r\n"+
				"Proxy error: %s",
			len(err.Error()), err.Error(),
		)
		io.WriteString(conn, errorResponse)
	}

	proxy.OnRequest().DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			return r, nil
		},
	)

	fmt.Println("[GALILEU] Proxy MITM Ativo na porta " + fmt.Sprintf(":%d", port) + "...")
	shutdownChan = make(chan struct{})

	// ✅ CORREÇÃO: Adicionar timeouts ao servidor HTTP
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
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

	baseTLSConfigFunc := goproxy.TLSConfigFromCA(&ca)

	forceHTTP1TLS := func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
		tlsConf, err := baseTLSConfigFunc(host, ctx)
		if err != nil {
			return nil, err
		}
		tlsConf.NextProtos = []string{"http/1.1"}
		return tlsConf, nil
	}

	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: forceHTTP1TLS}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: forceHTTP1TLS}
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

	if contents, ok := data["contents"].([]interface{}); ok {
		info.messageCount += len(contents)
		for _, c := range contents {
			if content, ok := c.(map[string]interface{}); ok {
				if role, ok := content["role"].(string); ok && role == "system" {
					info.hasSystemPrompt = true
				}
			}
		}
	}

	if genConfig, ok := data["generationConfig"].(map[string]interface{}); ok {
		if _, ok := genConfig["stopSequences"]; ok {
			// Gemini-specific config detected
		}
	}

	return info
}
