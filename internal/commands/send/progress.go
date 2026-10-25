package send

import (
	"os"

	"github.com/cheggaaa/pb/v3"
	"golang.org/x/term"
)

func startProgress(total int) *pb.ProgressBar {
	bar := pb.New(total).SetWriter(os.Stderr).Set(pb.CleanOnFinish, true)
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		// IDE consoles and redirected output cannot redraw the bar in place.
		// Static mode suppresses automatic output; the command prints its summary.
		bar.Set(pb.Static, true)
	}
	return bar.Start()
}
