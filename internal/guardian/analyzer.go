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
	compiledPatterns []PatternInfo
	redaction        []byte
}

func NewAnalyzer() *Analyzer {
	patterns := []PatternInfo{
		{PatternType: "openai_key", Regex: regexp.MustCompile(`(sk-[a-zA-Z0-9]{20,})`)},
		{PatternType: "openai_project_key", Regex: regexp.MustCompile(`(sk-proj-[a-zA-Z0-9.-]{20,})`)},
		{PatternType: "anthropic_key", Regex: regexp.MustCompile(`(sk-ant-[a-zA-Z0-9.-]{20,})`)},
		{PatternType: "google_key", Regex: regexp.MustCompile(`(AIzaSy[a-zA-Z0-9_-]{35})`)},
		{PatternType: "github_token", Regex: regexp.MustCompile(`(ghp_[a-zA-Z0-9]{36})`)},
		{PatternType: "slack_token", Regex: regexp.MustCompile(`(xox[baprs]-[a-zA-Z0-9]{10,})`)},
		{PatternType: "aws_access_key", Regex: regexp.MustCompile(`(AKIA[0-9A-Z]{16})`)},
		{PatternType: "generic_api_key", Regex: regexp.MustCompile(`(bearer\s+[a-zA-Z0-9.-]{20,})`)},
		{PatternType: "aws_secret_key", Regex: regexp.MustCompile(`(wJalr[a-zA-Z0-9/+=]{30,})`)},
		{PatternType: "generic_api_key", Regex: regexp.MustCompile(`(api_key[a-zA-Z0-9_]{20,})`)},
	}

	return &Analyzer{
		compiledPatterns: patterns,
		redaction:        []byte("[REDACTED_BY_GALILEU]"),
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
			result.DetectedPatterns = append(result.DetectedPatterns, p.PatternType)
			result.PatternCount += len(matches)
			result.RedactionPositions = append(result.RedactionPositions, "body")
			result.Result = p.Regex.ReplaceAll(result.Result, a.redaction)
		}
	}

	return result
}
