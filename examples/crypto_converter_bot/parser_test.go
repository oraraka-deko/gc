package main

import (
	"testing"
)

func TestParseConversionQuery_UserExamples(t *testing.T) {
	testCases := []struct {
		input        string
		expectedAmt  float64
		expectedFrom string
		expectedTo   string
	}{
		{"1 ton", 1.0, "TON", "USD"},
		{"1 TOn", 1.0, "TON", "USD"},
		{"1Ton", 1.0, "TON", "USD"},
		{"1grAm", 1.0, "TON", "USD"},
		{"gram1", 1.0, "TON", "USD"},
		{"ton1", 1.0, "TON", "USD"},
		{"gram   1", 1.0, "TON", "USD"},
		{"gram", 1.0, "TON", "USD"},
		{"ton", 1.0, "TON", "USD"},
		{"usdt", 1.0, "USDT", "USD"},
		{"tron 500", 500.0, "TRX", "USD"},
		{"tron ton", 1.0, "TRX", "TON"},
		{"usdt tron", 1.0, "USDT", "TRX"},
		{"tron usdt", 1.0, "TRX", "USDT"},
		{"500ton tron", 500.0, "TON", "TRX"},
		{"500 ton to tron", 500.0, "TON", "TRX"},
		{"how much is 100 gram in tron", 100.0, "TON", "TRX"},
		{"/price 2.5 ton", 2.5, "TON", "USD"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			q, ok := ParseConversionQuery(tc.input)
			if !ok {
				t.Fatalf("failed to parse query for input %q", tc.input)
			}
			if q.Amount != tc.expectedAmt {
				t.Errorf("input %q: expected amount %f, got %f", tc.input, tc.expectedAmt, q.Amount)
			}
			if q.FromCurrency != tc.expectedFrom {
				t.Errorf("input %q: expected from %s, got %s", tc.input, tc.expectedFrom, q.FromCurrency)
			}
			if q.ToCurrency != tc.expectedTo {
				t.Errorf("input %q: expected to %s, got %s", tc.input, tc.expectedTo, q.ToCurrency)
			}
		})
	}
}

func TestParseConversionQuery_NonQueries(t *testing.T) {
	nonQueries := []string{
		"hello world",
		"how are you doing",
		"just testing",
		"123456",
		"",
	}

	for _, input := range nonQueries {
		if q, ok := ParseConversionQuery(input); ok {
			t.Errorf("expected false for input %q, got %+v", input, q)
		}
	}
}
