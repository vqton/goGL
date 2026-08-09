package cash

import "strconv"

// FormatVNDMinor renders an int64 minor-unit amount with Vietnamese thousands
// separators, e.g. 1234567 -> "1.234.567". Single formatting implementation
// shared by the application service (notifier bodies) and print rendering.
func FormatVNDMinor(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	if neg {
		s = "-" + s
	}
	return s
}
