package components

import (
	"fmt"

	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// StepIndicator renders a step progress indicator like "● ● ○  Step 2/3"
func StepIndicator(current, total int) string {
	var dots string
	for i := 1; i <= total; i++ {
		if i == current {
			dots += styles.Violet.Render("●") + " "
		} else if i < current {
			dots += styles.Green.Render("●") + " "
		} else {
			dots += styles.Dim.Render("○") + " "
		}
	}
	return fmt.Sprintf("  %s %s", dots, styles.Dim.Render(fmt.Sprintf("Step %d/%d", current, total)))
}
