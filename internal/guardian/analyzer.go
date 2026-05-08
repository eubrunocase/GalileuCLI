package guardian

import (
	"regexp"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bufferWrapper{data: make([]byte, 0, 4096)}
	},
}

type bufferWrapper struct {
	data []byte
}

type PatternInfo struct {
	PatternType string
	Regex       *regexp.Regexp
}

type AnalysisResult struct {
	Modified           bool
	Result             []byte
	DetectedPatterns   []string
	PatternCount       int
	RedactionPositions []string
}

type Analyzer struct {
	compiledPatterns []CompiledPattern
	DryRun           bool
}

// NewAnalyzer cria um Analyzer com os padrões fornecidos externamente.
// Os padrões são compilados uma vez em LoadConfig e reutilizados aqui.
func NewAnalyzer(patterns []CompiledPattern) *Analyzer {
	return &Analyzer{
		compiledPatterns: patterns,
	}
}

func (a *Analyzer) Analyze(data []byte) AnalysisResult {
	result := AnalysisResult{
		DetectedPatterns:   make([]string, 0),
		RedactionPositions: make([]string, 0),
	}

	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	result.Result = dataCopy

	for _, p := range a.compiledPatterns {
		matches := p.Regex.FindAllIndex(result.Result, -1)
		if len(matches) > 0 {
			result.Modified = true
			result.DetectedPatterns = append(result.DetectedPatterns, p.Name)
			result.PatternCount += len(matches)
			result.RedactionPositions = append(result.RedactionPositions, "body")
			if !a.DryRun {
				result.Result = p.Regex.ReplaceAll(result.Result, []byte(p.Label))
			}
		}
	}

	return result
}
