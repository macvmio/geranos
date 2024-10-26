package transporter

import (
	"fmt"
	"github.com/macvmio/geranos/pkg/bitarray"
	"github.com/macvmio/geranos/pkg/dirimage"
)

type ProgressUpdate dirimage.ProgressUpdate

func PrintProgress(progress <-chan ProgressUpdate) {
	const maxSize = 800
	ba := bitarray.New(maxSize)
	updateProgress := func(progress int64) {
		ba.Fill(int(progress))
		fmt.Printf("\rProgress: %s %d%%", ba, progress/8)
	}
	last := int64(0)
	for p := range progress {
		current := maxSize * p.BytesProcessed / p.BytesTotal
		if current != last {
			updateProgress(current)
		}
		last = current
	}
	fmt.Printf("\n")
}
