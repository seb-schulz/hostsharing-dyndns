package main

import (
	"crypto/subtle"

	"golang.org/x/crypto/argon2"
)

type hashedPassword struct {
	key     []byte
	salt    []byte
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

func (p *hashedPassword) isValid(origPasswd []byte) bool {
	key := argon2.IDKey(origPasswd, p.salt, p.time, p.memory, p.threads, p.keyLen)
	return subtle.ConstantTimeCompare(key, p.key) == 1
}
