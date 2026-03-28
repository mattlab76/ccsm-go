package components

import (
	"testing"
)

func TestSparkline_Empty(t *testing.T) {
	result := RenderSparkline(nil, 20)
	if result != "" {
		t.Errorf("empty sparkline should be empty, got %q", result)
	}
}

func TestSparkline_SingleValue(t *testing.T) {
	data := []SparklineData{{Label: "01-01", Value: 100}}
	result := RenderSparkline(data, 20)
	if result == "" {
		t.Error("single value sparkline should not be empty")
	}
}

func TestSparkline_MultipleValues(t *testing.T) {
	data := []SparklineData{
		{Label: "01-01", Value: 10},
		{Label: "01-02", Value: 50},
		{Label: "01-03", Value: 100},
		{Label: "01-04", Value: 30},
		{Label: "01-05", Value: 0},
	}
	result := RenderSparkline(data, 20)
	if result == "" {
		t.Error("sparkline should not be empty")
	}
}

func TestSparkline_AllZeros(t *testing.T) {
	data := []SparklineData{
		{Label: "a", Value: 0},
		{Label: "b", Value: 0},
	}
	result := RenderSparkline(data, 20)
	if result == "" {
		t.Error("all-zero sparkline should still render")
	}
}

func TestSparkline_MaxWidth(t *testing.T) {
	data := make([]SparklineData, 50)
	for i := range data {
		data[i] = SparklineData{Label: "x", Value: int64(i)}
	}
	// With maxWidth=10, should only show last 10 values.
	result := RenderSparkline(data, 10)
	if result == "" {
		t.Error("sparkline should not be empty")
	}
}

func TestSparklineWithLabels(t *testing.T) {
	data := []SparklineData{
		{Label: "03-20", Value: 100},
		{Label: "03-21", Value: 200},
		{Label: "03-22", Value: 150},
	}
	result := RenderSparklineWithLabels(data, 20)
	if result == "" {
		t.Error("sparkline with labels should not be empty")
	}
}

func TestStepIndicator(t *testing.T) {
	result := StepIndicator(1, 3)
	if result == "" {
		t.Error("step indicator should not be empty")
	}
	result2 := StepIndicator(3, 3)
	if result2 == "" {
		t.Error("step indicator at last step should not be empty")
	}
}
