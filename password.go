package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/argon2"
	"gopkg.in/yaml.v3"
)

func argonPasswordValidator(key []byte, salt []byte, time, memory uint32, threads uint8, keyLen uint32) passwordValidator {
	return func(origPasswd []byte) bool {
		unverifiedKey := argon2.IDKey(origPasswd, salt, time, memory, threads, keyLen)
		return subtle.ConstantTimeCompare(key, unverifiedKey) == 1
	}
}

var (
	saltLength   uint16
	passwdLength uint16
	time         uint32
	memory       uint32
	threads      uint8
	keyLen       uint32
)

func init() {
	generatePasswordCmd.Flags().Uint16VarP(&saltLength, "salt", "s", 16, "byte size of generated salt")
	generatePasswordCmd.Flags().Uint16VarP(&passwdLength, "password", "p", 32, "byte size of generated password")
	generatePasswordCmd.Flags().Uint32Var(&time, "time", 1, "argon2id time parameter")
	generatePasswordCmd.Flags().Uint32VarP(&memory, "memory", "m", 64*1024, "argon2id memory parameter")
	generatePasswordCmd.Flags().Uint8Var(&threads, "threads", 4, "argon2id threads parameter")
	generatePasswordCmd.Flags().Uint32Var(&keyLen, "key-length", 32, "argon2id key length parameter")
}

var generatePasswordCmd = &cobra.Command{
	Use:     "generatePassword",
	Aliases: []string{"genpasswd", "gen"},
	Short:   "generate random password and print relevant parameters",
	RunE: func(cmd *cobra.Command, args []string) error {
		salt := make([]byte, saltLength)
		_, err := rand.Read(salt)
		if err != nil {
			return err
		}

		passwd := make([]byte, passwdLength)
		_, err = rand.Read(passwd)
		if err != nil {
			return err
		}

		key := argon2.IDKey(passwd, salt, time, memory, threads, keyLen)

		config, err := yaml.Marshal(struct {
			Key     string
			Salt    string
			Time    uint32
			Memory  uint32
			Threads uint8
			KeyLen  uint32
		}{
			Key:     base64.URLEncoding.EncodeToString(key),
			Salt:    base64.URLEncoding.EncodeToString(salt),
			Time:    time,
			Memory:  memory,
			Threads: threads,
			KeyLen:  keyLen,
		})
		if err != nil {
			return err
		}

		fmt.Printf("%s\n%s\n", config, base64.URLEncoding.EncodeToString(passwd))
		return nil
	},
}
