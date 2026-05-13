package screens

import "fmt"

// stringOfChar creates a string of n repeated characters.
func stringOfChar(c byte, n int) string {
	if n <= 0 {
		return ""
	}
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = c
	}
	return string(buf)
}

// itoa converts int to string.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	return fmt.Sprintf("%d", n)
}

// itoa64 converts int64 to string.
func itoa64(n int64) string {
	return fmt.Sprintf("%d", n)
}

// formatBytes formats bytes to human-readable string.
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(bytes)
	var i int
	for i = 0; i < len(units)-1 && size >= 1024; i++ {
		size /= 1024
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}
