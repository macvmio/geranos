package transporter

import (
	"fmt"
	"github.com/tomekjarosik/geranos/pkg/layout"
)

func List(opt ...Option) error {
	opts := makeOptions(opt...)
	lm := layout.NewMapper(opts.imagesPath)
	props, err := lm.List()
	if err != nil {
		return fmt.Errorf("unable to list images: %w", err)
	}
	// Print header
	fmt.Printf("%-50s %-15s %-15s %-10s\n", "REPOSITORY", "TAG", "SIZE", "DISK USAGE")

	for _, p := range props {
		fmt.Printf("%-50s %-15s %-15s %-10s\n", p.Ref.Context(), p.Ref.Identifier(),
			fmt.Sprintf("%d", p.Size), p.DiskUsage)
	}
	return nil
}
