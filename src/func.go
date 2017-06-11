// func.go

package main

import (
	"crypto/rand"
	"os"
)

//------------------------------------------------------------------------------

func generateRandomUint64() (rui uint64) {

	// Generates a cryptographically secure pseudorandom uint64 Number.

	var qidBA []byte
	var err error

	qidBA = make([]byte, 8)

	_, err = rand.Read(qidBA)
	if err != nil {
		panic(err) //
		os.Exit(1)
	}

	rui = (uint64(qidBA[0]) << 56) +
		(uint64(qidBA[1]) << 48) +
		(uint64(qidBA[2]) << 40) +
		(uint64(qidBA[3]) << 32) +
		(uint64(qidBA[4]) << 24) +
		(uint64(qidBA[5]) << 16) +
		(uint64(qidBA[6]) << 8) +
		uint64(qidBA[7])

	return rui
}

//------------------------------------------------------------------------------

func generateRandomUint32() (rui uint32) {

	// Generates a cryptographically secure pseudorandom uint32 Number.

	var qidBA []byte
	var err error

	qidBA = make([]byte, 4)

	_, err = rand.Read(qidBA)
	if err != nil {
		panic(err) //
		os.Exit(1)
	}

	rui = (uint32(qidBA[0]) << 24) +
		(uint32(qidBA[1]) << 16) +
		(uint32(qidBA[2]) << 8) +
		uint32(qidBA[3])

	return rui
}

//------------------------------------------------------------------------------

func generateRandomUint8() (rui uint8) {

	// Generates a cryptographically secure pseudorandom uint8 Number.

	var qidBA []byte
	var err error

	qidBA = make([]byte, 1)

	_, err = rand.Read(qidBA)
	if err != nil {
		panic(err) //
		os.Exit(1)
	}

	rui = uint8(qidBA[0])

	return rui
}

//------------------------------------------------------------------------------
