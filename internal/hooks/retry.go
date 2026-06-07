package hooks

import "time"

func retryDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return 0
	}
	return time.Duration(attempt-1) * 10 * time.Millisecond
}
