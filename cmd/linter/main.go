package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/Gustik/shortener/cmd/linter/analyzer"
)

func main() {
	singlechecker.Main(analyzer.Analyzer)
}
