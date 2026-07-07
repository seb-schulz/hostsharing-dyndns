package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestBase64StringToBytesHookFunc(t *testing.T) {
	hook, ok := base64StringToBytesHookFunc().(func(reflect.Type, reflect.Type, interface{}) (interface{}, error))
	if !ok {
		t.Fatal("hook is not a func with the expected signature")
	}
	for _, testCase := range []struct {
		name     string
		from     reflect.Type
		to       reflect.Type
		input    interface{}
		expected interface{}
	}{
		{
			name:     "non-string source passes through",
			from:     reflect.TypeOf(0),
			to:       reflect.TypeOf([]byte{}),
			input:    42,
			expected: 42,
		},
		{
			name:     "non-bytes target passes through",
			from:     reflect.TypeOf(""),
			to:       reflect.TypeOf(""),
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "valid URL base64 decodes",
			from:     reflect.TypeOf(""),
			to:       reflect.TypeOf([]byte{}),
			input:    "aGVsbG8=", // padded form; matches base64.URLEncoding output
			expected: []byte("hello"),
		},
		{
			name:     "invalid base64 falls back to original string",
			from:     reflect.TypeOf(""),
			to:       reflect.TypeOf([]byte{}),
			input:    "!!!not-base64!!!",
			expected: "!!!not-base64!!!",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := hook(testCase.from, testCase.to, testCase.input)
			if err != nil {
				t.Fatalf("hook returned error: %v", err)
			}
			if !reflect.DeepEqual(got, testCase.expected) {
				t.Errorf("got %v (%T), want %v (%T)", got, got, testCase.expected, testCase.expected)
			}
		})
	}
}

// chdirTempConfig drops a .hostsharing-dyndns.conf into a temp dir and
// chdirs there, since loadServerConfig reads config from the working dir.
func chdirTempConfig(t *testing.T, yaml string) func() {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/.hostsharing-dyndns.conf"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := os.Chdir(prev); err != nil {
			t.Logf("could not restore cwd: %v", err)
		}
	}
}

func TestLoadServerConfig(t *testing.T) {
	valid := `
UpdaterHandler:
  User: alice
  Filename: /tmp/zone.txt
  DomainSubpart: HOME.dyndns.example.com
  Password:
    Key: AAECAwQFBgcICQoLDA0ODw==
    Salt: AAECAwQFBgcICQoLDA0ODw==
    Time: 1
    Memory: 64
    Threads: 4
    KeyLen: 32
Logger:
  Enabled: true
`

	for _, testCase := range []struct {
		name       string
		yaml       string
		wantErrSub []string
	}{
		{"valid", valid, nil},
		{"missing user", strings.Replace(valid, "User: alice", `User: ""`, 1), []string{"undefined user"}},
		{"missing filename", strings.Replace(valid, "Filename: /tmp/zone.txt", `Filename: ""`, 1), []string{"undefined filename for zonefile"}},
		{"missing domain subpart", strings.Replace(valid, "DomainSubpart: HOME.dyndns.example.com", `DomainSubpart: ""`, 1), []string{"undefined domain subpart"}},
		{"short key", strings.Replace(valid, "Key: AAECAwQFBgcICQoLDA0ODw==", "Key: AA==", 1), []string{"undefined/short password key"}},
		{"short salt", strings.Replace(valid, "Salt: AAECAwQFBgcICQoLDA0ODw==", "Salt: AA==", 1), []string{"undefined/short password salt"}},
		{
			// All five validators must trip; errors.Join aggregates them.
			"all five missing",
			strings.NewReplacer(
				"User: alice", `User: ""`,
				"Filename: /tmp/zone.txt", `Filename: ""`,
				"DomainSubpart: HOME.dyndns.example.com", `DomainSubpart: ""`,
				"Key: AAECAwQFBgcICQoLDA0ODw==", "Key: AA==",
				"Salt: AAECAwQFBgcICQoLDA0ODw==", "Salt: AA==",
			).Replace(valid),
			[]string{
				"undefined user",
				"undefined filename for zonefile",
				"undefined domain subpart",
				"undefined/short password key",
				"undefined/short password salt",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			defer chdirTempConfig(t, testCase.yaml)()

			cfg, err := loadServerConfig()
			if testCase.wantErrSub == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if cfg.UpdaterHandler.User != "alice" {
					t.Errorf("user = %q, want %q", cfg.UpdaterHandler.User, "alice")
				}
				return
			}
			if err == nil {
				t.Fatalf("expected errors containing %v, got nil", testCase.wantErrSub)
			}
			for _, want := range testCase.wantErrSub {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q\nfull error:\n%s", err.Error(), want, err.Error())
				}
			}
		})
	}
}
