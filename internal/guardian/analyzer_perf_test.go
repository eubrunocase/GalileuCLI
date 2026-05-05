package guardian

import (
	"math"
	"sort"
	"testing"
	"time"
)

func TestAnalyzerLatency(t *testing.T) {
	payloads := loadTestPayloads()
	patterns := getTestPatterns()
	analyzer := NewAnalyzer(patterns)

	const iterations = 1000
	latencies := make([]int64, iterations*len(payloads))

	for i := 0; i < iterations; i++ {
		for j, payload := range payloads {
			before := sinceNano()
			analyzer.Analyze([]byte(payload))
			latencies[i*len(payloads)+j] = sinceNano() - before
		}
	}

	stats := calculateStats(latencies)

	t.Logf("=== Latência em Nanosegundos ===")
	t.Logf("Total de operações: %d", len(latencies))
	t.Logf("Média: %.2f ns (%.2f µs)", stats.mean, stats.mean/1e3)
	t.Logf("Min: %d ns", stats.min)
	t.Logf("Max: %d ns", stats.max)
	t.Logf("P50: %d ns (%.2f µs)", stats.p50, float64(stats.p50)/1e3)
	t.Logf("P95: %d ns (%.2f µs)", stats.p95, float64(stats.p95)/1e3)
	t.Logf("P99: %d ns (%.2f µs)", stats.p99, float64(stats.p99)/1e3)

	nsPerOp := stats.mean
	if nsPerOp > 3e6 {
		t.Errorf("Latência muito alta: %.2f ms por operação", nsPerOp/1e6)
	}

	opsPerSec := 1e9 / nsPerOp
	t.Logf("Throughput estimado: %.0f ops/s", opsPerSec)
}

func TestAnalyzerThroughput(t *testing.T) {
	payloads := loadTestPayloads()
	patterns := getTestPatterns()
	analyzer := NewAnalyzer(patterns)

	const iterations = 100
	totalPayloads := iterations * len(payloads)

	before := sinceNano()
	for i := 0; i < iterations; i++ {
		for _, payload := range payloads {
			analyzer.Analyze([]byte(payload))
		}
	}
	elapsed := sinceNano() - before

	avgNs := float64(elapsed) / float64(totalPayloads)
	opsPerSec := 1e9 / avgNs
	mbPerSec := float64(totalPayloads*1000) / (float64(elapsed) / 1e6)

	t.Logf("=== Throughput ===")
	t.Logf("Total de requisições processadas: %d", totalPayloads)
	t.Logf("Tempo total: %.2f ms", float64(elapsed)/1e6)
	t.Logf("Média por requisição: %.2f µs", avgNs/1e3)
	t.Logf("Throughput: %.0f req/s", opsPerSec)
	t.Logf("Throughput: %.2f MB/s", mbPerSec)
}

func loadTestPayloads() []string {
	return []string{
		`{"model": "gpt-4o", "messages": [{"role": "user", "content": "Hello"}], "stream": false}`,
		`{"model": "gpt-4-turbo", "messages": [{"role": "user", "content": "Write code"}], "temperature": 0.7}`,
		`{"model": "claude-3-opus-20240229", "max_tokens": 4096, "messages": []}`,
		`{"model": "gemini-pro", "messages": [{"role": "user", "parts": [{"text": "Hi"}]}]}`,
		`{"key": "sk-proj-test1234567890abcdefghijklmn", "model": "gpt-4"}`,
		`{"user_id": 12345, "timestamp": "2024-01-01T00:00:00Z"}`,
		`{"array": [1, 2, 3, 4, 5], "nested": {"a": 1, "b": 2}}`,
	}
}

type latencyStats struct {
	mean float64
	min  int64
	max  int64
	p50  int64
	p95  int64
	p99  int64
}

func calculateStats(latencies []int64) latencyStats {
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	n := len(latencies)
	var sum int64
	for _, l := range latencies {
		sum += l
	}

	return latencyStats{
		mean: float64(sum) / float64(n),
		min:  latencies[0],
		max:  latencies[n-1],
		p50:  latencies[n/2],
		p95:  latencies[int(math.Floor(float64(n)*0.95))],
		p99:  latencies[int(math.Floor(float64(n)*0.99))],
	}
}

func sinceNano() int64 {
	return time.Now().UnixNano()
}
