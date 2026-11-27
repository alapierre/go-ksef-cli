package tokens

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"go-ksef-cli/internal/app"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/alecthomas/kong"
	"github.com/zalando/go-keyring"
)

func TestGenerateRequest(t *testing.T) {
	for _, tc := range []struct {
		name  string
		extra []string
		valid bool
	}{
		{"defaults", nil, true},
		{"future end only", []string{"--valid-to", "2027-05-01T09:30:00+02:00"}, true},
		{"end before start", []string{"--valid-from", "2027-05-02T00:00:00Z", "--valid-to", "2027-05-01T00:00:00Z"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var cmd GenerateCmd
			parser, err := kong.New(&cmd)
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"--out", "successor.token", "-i", "1111111111", "--description", "Rotation test", "--permission", "InvoiceRead", "--permission", "InvoiceWrite"}
			if _, err = parser.Parse(append(args, tc.extra...)); err != nil {
				t.Fatal(err)
			}
			req, err := cmd.request()
			if !tc.valid {
				if err == nil {
					t.Fatal("invalid time range accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(req.Permissions) != 2 || req.Permissions[0] != api.TokenPermissionTypeInvoiceRead ||
				req.Permissions[1] != api.TokenPermissionTypeInvoiceWrite || req.ValidFrom.IsSet() {
				t.Fatalf("unexpected request: %+v", req)
			}
			end, set := req.ValidTo.Get()
			if set != (cmd.ValidTo != nil) || (set && end.Format(time.RFC3339) != "2027-05-01T09:30:00+02:00") {
				t.Fatalf("validTo = %v, %t", end, set)
			}
		})
	}
}

type fakeTokenIssuer struct {
	calls int
	err   error
}

func (f *fakeTokenIssuer) GenerateKsefToken(context.Context, api.TokensRev2PostReq) (*ksef.GeneratedKsefToken, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &ksef.GeneratedKsefToken{Token: "candidate-secret", ReferenceNumber: "test-reference"}, nil
}

func TestGeneratePlaintextTokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "candidate.token")
	cmd := GenerateCmd{Out: path, Plaintext: true}
	client := &fakeTokenIssuer{err: errors.New("API unavailable")}
	ctx := context.Background()
	req := api.TokensRev2PostReq{}
	if _, err := cmd.generateToFile(ctx, client, req, "TEST"); !errors.Is(err, client.err) {
		t.Fatalf("API failure = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file after failed issuance: %v", err)
	}

	client.err = nil
	issued, err := cmd.generateToFile(ctx, client, req, "TEST")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cmd.generateToFile(ctx, client, req, "TEST"); !errors.Is(err, os.ErrExist) || client.calls != 2 {
		t.Fatalf("existing path: err=%v, generation calls=%d", err, client.calls)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != issued.Token || issued.ReferenceNumber != "test-reference" {
		t.Fatal("issued token was not saved correctly")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Errorf("token file mode = %v", info.Mode())
	}
}

func TestGenerateEncryptsByDefault(t *testing.T) {
	keyring.MockInit()
	client := &fakeTokenIssuer{}
	cmd := GenerateCmd{Out: filepath.Join(t.TempDir(), "candidate.enc"), Identifier: "1111111111"}
	generate := func() (*ksef.GeneratedKsefToken, error) {
		return cmd.generateToFile(context.Background(), client, api.TokensRev2PostReq{}, "TEST")
	}
	if _, err := generate(); !errors.Is(err, app.ErrEncryptionKeyNotInitialized) || client.calls != 0 {
		t.Fatalf("missing key: err=%v, calls=%d", err, client.calls)
	}
	if err := app.InitEncryption(); err != nil {
		t.Fatal(err)
	}
	issued, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(cmd.Out)
	if err != nil || bytes.Contains(data, []byte(issued.Token)) {
		t.Fatal("expected encrypted output")
	}
	got, err := app.ReadEncryptedAuthTokenFile(cmd.Out, "test", cmd.Identifier)
	if err != nil || got != issued.Token {
		t.Fatalf("candidate did not round trip: %v", err)
	}
	for _, tc := range []struct{ env, nip string }{{"DEMO", cmd.Identifier}, {"TEST", "2222222222"}} {
		if _, err := app.ReadEncryptedAuthTokenFile(cmd.Out, tc.env, tc.nip); !errors.Is(err, app.ErrAuthTokenDecrypt) {
			t.Fatalf("wrong context: %v", err)
		}
	}
	if _, err := generate(); !errors.Is(err, os.ErrExist) || client.calls != 1 {
		t.Fatalf("existing encrypted file: err=%v, calls=%d", err, client.calls)
	}
	data[len(data)-1] ^= 1
	if err := os.WriteFile(cmd.Out, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := app.ReadEncryptedAuthTokenFile(cmd.Out, "TEST", cmd.Identifier); !errors.Is(err, app.ErrAuthTokenDecrypt) {
		t.Fatalf("tampered file: %v", err)
	}
}
