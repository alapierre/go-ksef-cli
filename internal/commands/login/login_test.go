package login

import (
	"testing"

	"github.com/alecthomas/kong"
)

func TestLoginCmdFlagParsing(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantRaw     bool
		wantPrint   bool
		wantNoStore bool
		wantNip     string
		wantToken   string
	}{
		{
			name:        "short raw flag",
			args:        []string{"-i", "1234567890", "-r"},
			wantRaw:     true,
			wantPrint:   false,
			wantNoStore: false,
			wantNip:     "1234567890",
		},
		{
			name:        "long raw flag",
			args:        []string{"-i", "1234567890", "--raw"},
			wantRaw:     true,
			wantPrint:   false,
			wantNoStore: false,
			wantNip:     "1234567890",
		},
		{
			name:        "print and raw flags together",
			args:        []string{"-i", "1234567890", "-p", "--raw", "-n"},
			wantRaw:     true,
			wantPrint:   true,
			wantNoStore: true,
			wantNip:     "1234567890",
		},
		{
			name:        "default without raw",
			args:        []string{"-i", "1234567890", "-p", "-t", "some-token"},
			wantRaw:     false,
			wantPrint:   true,
			wantNoStore: false,
			wantNip:     "1234567890",
			wantToken:   "some-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd Cmd
			parser, err := kong.New(&cmd)
			if err != nil {
				t.Fatalf("kong.New failed: %v", err)
			}
			_, err = parser.Parse(tt.args)
			if err != nil {
				t.Fatalf("parser.Parse(%v) failed: %v", tt.args, err)
			}
			if cmd.Raw != tt.wantRaw {
				t.Errorf("Raw = %v, want %v", cmd.Raw, tt.wantRaw)
			}
			if cmd.PrintSessionToken != tt.wantPrint {
				t.Errorf("PrintSessionToken = %v, want %v", cmd.PrintSessionToken, tt.wantPrint)
			}
			if cmd.NoStore != tt.wantNoStore {
				t.Errorf("NoStore = %v, want %v", cmd.NoStore, tt.wantNoStore)
			}
			if cmd.Identifier != tt.wantNip {
				t.Errorf("Identifier = %q, want %q", cmd.Identifier, tt.wantNip)
			}
			if cmd.Token != tt.wantToken {
				t.Errorf("Token = %q, want %q", cmd.Token, tt.wantToken)
			}
		})
	}
}
