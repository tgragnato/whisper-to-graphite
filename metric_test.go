package main

import (
	"testing"
	"time"
)

func TestMetricString(t *testing.T) {
	testCases := []struct {
		name     string
		metric   Metric
		expected string
	}{
		{
			name: "Basic metric",
			metric: Metric{
				Name:      "test.metric",
				Value:     "123.45",
				Timestamp: 1609459200, // 2021-01-01 00:00:00
			},
			expected: "test.metric 123.45 2021-01-01 00:00:00",
		},
		{
			name: "Empty values",
			metric: Metric{
				Name:      "",
				Value:     "",
				Timestamp: 1609459200,
			},
			expected: "  2021-01-01 00:00:00",
		},
		{
			name: "Zero timestamp",
			metric: Metric{
				Name:      "zero.time",
				Value:     "0",
				Timestamp: 0,
			},
			expected: "zero.time 0 1970-01-01 00:00:00",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.metric.String()
			if result != tc.expected {
				t.Errorf("String() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestNewMetric(t *testing.T) {
	name := "system.cpu.usage"
	value := "95.5"
	timestamp := time.Now().Unix()

	metric := NewMetric(name, value, timestamp)

	if metric.Name != name {
		t.Errorf("Expected Name to be %q, got %q", name, metric.Name)
	}
	if metric.Value != value {
		t.Errorf("Expected Value to be %q, got %q", value, metric.Value)
	}
	if metric.Timestamp != timestamp {
		t.Errorf("Expected Timestamp to be %d, got %d", timestamp, metric.Timestamp)
	}
}
