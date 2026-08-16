package masterdata

import "time"

// nowRFC3339 returns the current UTC time in RFC3339 (persisted format).
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
