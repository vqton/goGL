package cash

import (
	"strings"
)

var units = []string{"", "một", "hai", "ba", "bốn", "năm", "sáu", "bảy", "tám", "chín"}

var tens = []string{"", "", "hai mươi", "ba mươi", "bốn mươi", "năm mươi", "sáu mươi", "bảy mươi", "tám mươi", "chín mươi"}

var scales = []string{"", "nghìn", "triệu", "tỷ", "nghìn tỷ", "triệu tỷ", "tỷ tỷ"}

// AmountInWords renders a whole minor-unit amount as statutory Vietnamese
// number-in-words, e.g. 123045067 -> "một trăm hai mươi ba triệu không trăm
// bốn mươi lăm nghìn không trăm sáu mươi bảy đồng" (TT99/TT200 sample).
func AmountInWords(minor int64) string {
	if minor == 0 {
		return "không đồng"
	}
	prefix := ""
	if minor < 0 {
		prefix = "âm "
		minor = -minor
	}

	groups := make([]int, 0, 7)
	for minor > 0 {
		groups = append(groups, int(minor%1000))
		minor /= 1000
	}

	lead := len(groups) - 1
	parts := make([]string, 0, len(groups)*2)
	for i := lead; i >= 0; i-- {
		if groups[i] == 0 {
			continue
		}
		parts = append(parts, readGroup(groups[i], i != lead))
		if i > 0 {
			parts = append(parts, scales[i])
		}
	}
	return prefix + strings.Join(parts, " ") + " đồng"
}

func readGroup(g int, nonLead bool) string {
	h := g / 100
	r := g % 100
	hasHundreds := false

	w := make([]string, 0, 3)
	if h > 0 {
		w = append(w, units[h]+" trăm")
		hasHundreds = true
	} else if nonLead && r > 0 {
		w = append(w, "không trăm")
		hasHundreds = true
	}
	if r == 0 {
		return strings.Join(w, " ")
	}

	t := r / 10
	u := r % 10
	switch {
	case r < 10:
		if hasHundreds {
			w = append(w, "linh "+units[u])
		} else {
			w = append(w, units[u])
		}
	case r < 20:
		if u == 0 {
			w = append(w, "mười")
		} else if u == 5 {
			w = append(w, "mười lăm")
		} else {
			w = append(w, "mười "+units[u])
		}
	default:
		w = append(w, tens[t])
		switch u {
		case 1:
			w = append(w, "mốt")
		case 4:
			w = append(w, "tư")
		case 5:
			w = append(w, "lăm")
		default:
			if u > 0 {
				w = append(w, units[u])
			}
		}
	}
	return strings.Join(w, " ")
}

// RefPrefix returns the voucher-number prefix for the type: PT for a receipt
// (phiếu thu), PC for a payment (phiếu chi).
func (t VoucherType) RefPrefix() string {
	if t == VoucherReceive {
		return "PT"
	}
	return "PC"
}
