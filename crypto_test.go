package main

import (
	"crypto/sha256"
	"encoding/binary"
	"log"
	"math/rand"
	"testing"
	"time"
)

func TestSHA256(t *testing.T) {
	rand.Seed(time.Now().Unix())
	buf := make([]byte, binary.MaxVarintLen64)
	sum := make([]byte, 0)
	n := binary.PutUvarint(buf, rand.Uint64())
	sum = append(sum, buf[:n]...)
	c := 1

	numBytes := uint(250)
	log.Println("numBytes:", numBytes)
	for {
		// log.Printf("%x %d\n", sum, len(sum))
		sumBytes := sha256.Sum256([]byte(sum))
		// digits := ByteToNumDigits(sumBytes[:])
		// log.Println("number of digits:", digits)
		if ByteToNumDigits(sumBytes[:]) < numBytes {
			break
		}
		n = binary.PutUvarint(buf, rand.Uint64())
		sum = append(sum, buf[:n]...)
		if len(sum) > 1000 {
			sum = sum[len(sum)-1000:]
		}
		c++
		if c%1000000 == 0 {
			log.Println("trials:", c)
		}
	}
	log.Println("total trail:", c)
}
