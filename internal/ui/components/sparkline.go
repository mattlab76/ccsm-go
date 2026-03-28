package components

import (
	"strings"

	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// Sparkline renders a mini bar chart from a series of values.
// Uses Unicode block characters: ▁▂▃▄▅▆▇█
// Each value is represented by one character in the chart.
var blocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// SparklineData holds a label and value for one data point.
type SparklineData struct {
	Label string
	Value int64
}

// RenderSparkline creates a sparkline string from values.
// maxWidth limits the number of bars shown (most recent values).
func RenderSparkline(data []SparklineData, maxWidth int) string {
	if len(data) == 0 {
		return ""
	}
	// Limit to maxWidth most recent entries.
	start := 0
	if len(data) > maxWidth {
		start = len(data) - maxWidth
	}
	visible := data[start:]

	// Find max value.
	var maxVal int64
	for _, d := range visible {
		if d.Value > maxVal {
			maxVal = d.Value
		}
	}
	if maxVal == 0 {
		return strings.Repeat(string(blocks[0]), len(visible))
	}

	var b strings.Builder
	for _, d := range visible {
		idx := int(float64(d.Value) / float64(maxVal) * float64(len(blocks)-1))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		if idx < 0 {
			idx = 0
		}
		b.WriteRune(blocks[idx])
	}
	return styles.Violet.Render(b.String())
}

// RenderSparklineWithLabels renders a sparkline with start/end date labels.
func RenderSparklineWithLabels(data []SparklineData, maxWidth int) string {
	if len(data) == 0 {
		return ""
	}
	spark := RenderSparkline(data, maxWidth)

	start := 0
	if len(data) > maxWidth {
		start = len(data) - maxWidth
	}
	firstLabel := data[start].Label
	lastLabel := data[len(data)-1].Label

	return styles.Dim.Render(firstLabel) + " " + spark + " " + styles.Dim.Render(lastLabel)
}
