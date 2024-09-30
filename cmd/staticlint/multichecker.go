package main

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	analysis2 "github.com/Vidkin/metrics/pkg/analysis"
)

const (
	AnalyzerReceiverNames     = "ST1006"
	AnalyzerErrorShouldBeLast = "ST1008"
)

func getStaticCheckAnalizers() []*analysis.Analyzer {
	var checks []*analysis.Analyzer
	for _, v := range staticcheck.Analyzers {
		if strings.HasPrefix(v.Analyzer.Name, "SA") {
			checks = append(checks, v.Analyzer)
		}
	}
	for _, v := range stylecheck.Analyzers {
		if v.Analyzer.Name == AnalyzerReceiverNames || v.Analyzer.Name == AnalyzerErrorShouldBeLast {
			checks = append(checks, v.Analyzer)
		}
	}
	return checks
}

func getAnalysisAnalizers() []*analysis.Analyzer {
	res := make([]*analysis.Analyzer, 45)
	res[0] = asmdecl.Analyzer
	res[1] = assign.Analyzer
	res[2] = atomic.Analyzer
	res[3] = atomicalign.Analyzer
	res[4] = bools.Analyzer
	res[5] = buildssa.Analyzer
	res[6] = buildtag.Analyzer
	res[7] = cgocall.Analyzer
	res[8] = composite.Analyzer
	res[9] = copylock.Analyzer
	res[10] = ctrlflow.Analyzer
	res[11] = deepequalerrors.Analyzer
	res[12] = defers.Analyzer
	res[13] = directive.Analyzer
	res[14] = errorsas.Analyzer
	res[15] = fieldalignment.Analyzer
	res[16] = findcall.Analyzer
	res[17] = framepointer.Analyzer
	res[18] = httpresponse.Analyzer
	res[19] = ifaceassert.Analyzer
	res[20] = inspect.Analyzer
	res[21] = loopclosure.Analyzer
	res[22] = lostcancel.Analyzer
	res[23] = nilfunc.Analyzer
	res[24] = nilness.Analyzer
	res[25] = pkgfact.Analyzer
	res[26] = printf.Analyzer
	res[27] = reflectvaluecompare.Analyzer
	res[28] = shadow.Analyzer
	res[29] = shift.Analyzer
	res[30] = sigchanyzer.Analyzer
	res[31] = slog.Analyzer
	res[32] = sortslice.Analyzer
	res[33] = stdmethods.Analyzer
	res[34] = stringintconv.Analyzer
	res[35] = structtag.Analyzer
	res[36] = testinggoroutine.Analyzer
	res[37] = tests.Analyzer
	res[38] = timeformat.Analyzer
	res[39] = unmarshal.Analyzer
	res[40] = unreachable.Analyzer
	res[41] = unsafeptr.Analyzer
	res[42] = unusedresult.Analyzer
	res[43] = unusedwrite.Analyzer
	res[44] = usesgenerics.Analyzer

	return res
}

func main() {
	allChecks := make([]*analysis.Analyzer, 0)
	allChecks = append(allChecks, analysis2.ExitMainAnalyzer)
	allChecks = append(allChecks, getAnalysisAnalizers()...)
	allChecks = append(allChecks, getStaticCheckAnalizers()...)
	multichecker.Main(
		allChecks...,
	)
}
