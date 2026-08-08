package cash_test

import (
	"testing"

	"goGL/internal/domain/cash"
)

func TestAmountInWords(t *testing.T) {
	cases := []struct {
		minor int64
		want  string
	}{
		{0, "không đồng"},
		{123, "một trăm hai mươi ba đồng"},
		{100, "một trăm đồng"},
		{101, "một trăm linh một đồng"},
		{105, "một trăm linh năm đồng"},
		{110, "một trăm mười đồng"},
		{111, "một trăm mười một đồng"},
		{115, "một trăm mười lăm đồng"},
		{10, "mười đồng"},
		{15, "mười lăm đồng"},
		{20, "hai mươi đồng"},
		{21, "hai mươi mốt đồng"},
		{25, "hai mươi lăm đồng"},
		{54, "năm mươi tư đồng"},
		{99, "chín mươi chín đồng"},
		{1000, "một nghìn đồng"},
		{1000000, "một triệu đồng"},
		{1000000000, "một tỷ đồng"},
		{123045067, "một trăm hai mươi ba triệu không trăm bốn mươi lăm nghìn không trăm sáu mươi bảy đồng"},
		{2000050, "hai triệu không trăm năm mươi đồng"},
		{1005050, "một triệu không trăm linh năm nghìn không trăm năm mươi đồng"},
		{1000050, "một triệu không trăm năm mươi đồng"},
		{-250000, "âm hai trăm năm mươi nghìn đồng"},
	}
	for _, tc := range cases {
		if got := cash.AmountInWords(tc.minor); got != tc.want {
			t.Errorf("AmountInWords(%d) = %q, want %q", tc.minor, got, tc.want)
		}
	}
}
