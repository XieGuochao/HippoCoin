package host

import (
	"encoding/hex"
	"log"
	"testing"
	"time"
)

// Uint32ToBytes ...
func Uint32ToBytes(n uint32) []byte {
	return []byte{
		byte(n),
		byte(n >> 8),
		byte(n >> 16),
		byte(n >> 24),
	}
}

// Uint64ToBytes ...
func Uint64ToBytes(n uint64) []byte {
	return []byte{
		byte(n),
		byte(n >> 8),
		byte(n >> 16),
		byte(n >> 24),
		byte(n >> 32),
		byte(n >> 40),
		byte(n >> 48),
		byte(n >> 56),
	}
}

// ByteToUint32 ...
func ByteToUint32(b []byte) (n uint32) {
	if len(b) != 4 {
		return 0
	}
	return uint32(b[0]) + (uint32(b[1]) << 8) + (uint32(b[2]) << 16) + (uint32(b[3]) << 24)
}

// ByteToString ...
// Byte to hex string.
var ByteToString = hex.EncodeToString

// StringToByte ...
// String to hex byte.
var StringToByte = hex.DecodeString

// ByteToHexString ...
func ByteToHexString(bytes []byte) string {
	return hex.EncodeToString(bytes)
}

// ByteToNumDigits ...
// Calculate the number of digits of a byte slice.
func ByteToNumDigits(input []byte) (number uint) {
	number = 0
	startingIndex := 0
	for startingIndex = 0; startingIndex < len(input); startingIndex++ {
		if input[startingIndex] > 0 {
			break
		}
	}
	if startingIndex == len(input) {
		// no byte.
		return 0
	}
	number += uint(8 * (len(input) - startingIndex - 1))
	k := input[startingIndex]
	for k > 0 {
		k = k >> 1
		number++
	}
	return number
}

func assert(p bool) {
	if !p {
		panic(p)
	}
}

func assertT(p bool, t *testing.T) {
	if !p {
		t.Fail()
	}
}

func measureTime(funcName string) func() {
	start := time.Now()
	return func() {
		log.Printf("Time taken by %s function is %v \n", funcName, time.Since(start))
	}
}
