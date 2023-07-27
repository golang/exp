package p

// This test demonstrates correct handling of symbols
// in packages other than two being compared.
// See the lines in establishCorrespondence after
//   	if newn, ok := new.(*types.Named); ok

// both

// gofmt insists on grouping imports, so old and new
// must both have both imports.
import (
	"io"
	"text/tabwriter"
)

// old
var V io.Writer
var _ tabwriter.Writer

// new
// i V: changed from io.Writer to text/tabwriter.Writer
var V tabwriter.Writer
var _ io.Writer
