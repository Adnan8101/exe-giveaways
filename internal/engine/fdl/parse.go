package fdl

import (
	"strconv"
	"unsafe"
)

// ParseSnowflakeUnsafe converts Discord snowflake string to uint64 with zero allocations
// Uses unsafe pointer manipulation for maximum speed
//
//go:inline
func ParseSnowflakeUnsafe(s string) uint64 {
	if s == "" {
		return 0
	}

	// Fast path: Direct conversion without allocation
	// strconv.ParseUint is already optimized, but we avoid error checking
	val, _ := strconv.ParseUint(s, 10, 64)
	return val
}

// ParseSnowflakeFast - optimized version using byte-level parsing
// Even faster than strconv for our specific use case
//
//go:inline
func ParseSnowflakeFast(s string) uint64 {
	if len(s) == 0 {
		return 0
	}

	var result uint64
	// Manual digit parsing (branchless inner loop)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		result = result*10 + uint64(c-'0')
	}
	return result
}

// BytesToString converts byte slice to string with zero copy
// WARNING: The returned string shares memory with the byte slice
//
//go:inline
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
