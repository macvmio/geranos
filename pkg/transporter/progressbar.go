package transporter

import (
	"fmt"
	"github.com/mobileinf/geranos/pkg/dirimage"
	"strings"
)

type ProgressUpdate dirimage.ProgressUpdate

func PrintProgress(progress <-chan ProgressUpdate) {
	updateProgress := func(progress int64) {
		buffer := new(strings.Builder)
		fmt.Fprintf(buffer, "\rProgress: [%-100s] %d%%", strings.Repeat("=", int(progress)), progress)
		fmt.Print(buffer.String())
	}
	last := int64(0)
	for p := range progress {
		current := 100 * p.BytesProcessed / p.BytesTotal
		if current != last {
			updateProgress(current)
		}
		last = current
	}
	fmt.Printf("\n")
}
