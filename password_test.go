package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestArgonPasswordValidator(t *testing.T) {
	for _, testCase := range []struct {
		v              passwordValidator
		password       []byte
		expectedResult bool
	}{
		{
			argonPasswordValidator([]byte{230, 104, 139, 86, 35, 176, 125, 179, 79, 26, 88, 17, 178, 50, 28, 214, 27, 165, 105, 84, 225, 141, 44, 123, 62, 196, 70, 127, 108, 203, 144, 225}, []byte("123"), 1, 1, 1, 32),
			[]byte(".test."), true,
		}, {
			argonPasswordValidator([]byte{110, 229, 10, 51, 153, 202, 41, 137, 248, 79, 231, 236, 127, 187, 80, 94, 249, 57, 166, 194, 156, 43, 72, 188, 139, 201, 240, 81, 164, 31, 152, 176}, []byte("abc"), 1, 1, 1, 32),
			[]byte(".test."), true,
		}, {
			argonPasswordValidator([]byte{230, 104, 139, 86, 35, 176, 125, 179, 79, 26, 88, 17, 178, 50, 28, 214, 27, 165, 105, 84, 225, 141, 44, 123, 62, 196, 70, 127, 108, 203, 144, 225}, []byte("123"), 1, 1, 1, 32),
			[]byte("..test.."), false,
		},
	} {
		if got := testCase.v(testCase.password); got != testCase.expectedResult {
			t.Errorf("test result not as expected: %v instead of %v", got, testCase.expectedResult)
		}
	}
}

func TestGeneratePasswordCmd(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		saltLength    uint16
		passwdLength  uint16
		time          uint32
		memory        uint32
		threads       uint8
		keyLen        uint32
		wantInYAML    []string
		wantPasswdLen int // expected length of the printed password (RawURLEncoding)
	}{
		{
			name:          "defaults",
			saltLength:    16,
			passwdLength:  32,
			time:          1,
			memory:        64 * 1024,
			threads:       4,
			keyLen:        32,
			wantInYAML:    []string{"key:", "salt:", "time: 1", "memory: 65536", "threads: 4", "keylen: 32"},
			wantPasswdLen: 43, // 32 bytes -> 43 chars base64url (no padding)
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			prevStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = w
			t.Cleanup(func() { os.Stdout = prevStdout })

			done := make(chan string, 1)
			go func() {
				var buf bytes.Buffer
				_, _ = io.Copy(&buf, r)
				done <- buf.String()
			}()

			saltLength = testCase.saltLength
			passwdLength = testCase.passwdLength
			time = testCase.time
			memory = testCase.memory
			threads = testCase.threads
			keyLen = testCase.keyLen

			if err := generatePasswordCmd.RunE(generatePasswordCmd, nil); err != nil {
				t.Fatalf("RunE returned error: %v", err)
			}
			_ = w.Close()
			out := <-done

			parts := strings.SplitN(out, "\n\n", 2)
			if len(parts) != 2 {
				t.Fatalf("expected YAML + password separated by blank line, got %q", out)
			}
			yamlPart, passwd := parts[0], strings.TrimSpace(parts[1])

			for _, want := range testCase.wantInYAML {
				if !strings.Contains(strings.ToLower(yamlPart), want) {
					t.Errorf("YAML missing %q\nfull YAML:\n%s", want, yamlPart)
				}
			}
			if len(passwd) != testCase.wantPasswdLen {
				t.Errorf("password length = %d, want %d (got %q)", len(passwd), testCase.wantPasswdLen, passwd)
			}
		})
	}
}
