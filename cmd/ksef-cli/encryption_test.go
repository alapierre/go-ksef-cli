package main

import (
	"fmt"
	"testing"
)

func Test_decrypt(t *testing.T) {

	err := initEncryption()
	if err != nil {
		return
	}

	encrypted, err := crypt("Ala ma kota")
	if err != nil {
		return
	}

	fmt.Println(encrypted)

	plain, err := decrypt(encrypted)
	if err != nil {
		return
	}

	fmt.Println(plain)
}

func Test_storeAuthToken(t *testing.T) {

	storeAuthToken("alamakota", "6891152920", "test")
	token := loadAuthToken("6891152920", "test")

	fmt.Println(token)
}

func Test_storeSessionToken(t *testing.T) {

	storeSessionToken("alamakota", "test")
	token := loadSessionToken("test")

	fmt.Println(token)
}
