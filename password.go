package main

import (
	"crypto/subtle"

	"golang.org/x/crypto/argon2"
)

func argonPasswordValidator(key []byte, salt []byte, time, memory uint32, threads uint8, keyLen uint32) passwordValidator {
	return func(origPasswd []byte) bool {
		unverifiedKey := argon2.IDKey(origPasswd, salt, time, memory, threads, keyLen)
		return subtle.ConstantTimeCompare(key, unverifiedKey) == 1
	}
}
