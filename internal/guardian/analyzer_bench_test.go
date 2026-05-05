package guardian

import (
	"os"
	"testing"
)

func BenchmarkAnalyze(b *testing.B) {
	payload, err := os.ReadFile("../../galileu_audit.log")
	if err != nil {
		b.Fatal(err)
	}
	analyzer := NewAnalyzer(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(payload)
	}
}
