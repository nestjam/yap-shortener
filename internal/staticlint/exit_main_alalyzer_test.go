package staticlint

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestExitMainAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ExitMainAnalyzer, "./...")
}
