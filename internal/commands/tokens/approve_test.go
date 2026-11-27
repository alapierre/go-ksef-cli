package tokens

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go-ksef-cli/internal/apperrors"
	"go-ksef-cli/internal/config"

	"github.com/alecthomas/kong"
)

type approvalTransport func(*http.Request) (*http.Response, error)

func (f approvalTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestApproveWithSignedSession(t *testing.T) {
	for _, command := range []string{"approve card", "approve", "approve cert"} {
		t.Run(command, func(t *testing.T) {
			testApproveWithSignedSession(t, command, `{"code":200,"description":"success"}`, false)
		})
	}
}

func TestApproveDisplaysHintForAuthentication415(t *testing.T) {
	testApproveWithSignedSession(t, "approve", `{"code":415,"description":"Uwierzytelnianie zakończone niepowodzeniem"}`, true)
}

func testApproveWithSignedSession(t *testing.T, command, authStatus string, wantFailure bool) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("test signer uses a POSIX shell")
	}
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("KSEF_TOKEN", "unrelated-default-token")
	const ref = "20261106-EC-22ACCDF000-5CDA6D5392-B0"
	args := append(strings.Fields(command), "-i", "1111111111", ref)
	t.Setenv("SIG_PIN", "card-pin")
	t.Setenv("EXPECTED_BACKEND", "card")
	if command == "approve cert" {
		t.Setenv("EXPECTED_BACKEND", "keystore")
		t.Setenv("KSEF_KEY_PASSWORD", " key password with spaces ")
		cert := filepath.Join(dir, "auth cert.crt")
		key := filepath.Join(dir, "auth private.key")
		for _, path := range []string{cert, key} {
			if err := os.WriteFile(path, nil, 0600); err != nil {
				t.Fatal(err)
			}
		}
		args = append(args, "--cert", cert, "--key", key)
	} else {
		module := filepath.Join(dir, "driver with spaces.so")
		if err := os.WriteFile(module, nil, 0600); err != nil {
			t.Fatal(err)
		}
		args = append(args, "--module", module, "--slot", "2")
	}
	const script = `#!/bin/sh
set -eu
[ "$#" = 6 ] && [ "$1" = "$EXPECTED_BACKEND" ] || exit 42
case "$1" in
  card) [ "$2" = -d ] && [ "${3##*/}" = "driver with spaces.so" ] && [ "$4" = -s ] && [ "$5" = 2 ] && [ "$SIG_PIN" = card-pin ] || exit 43 ;;
  keystore) [ "$2" = -c ] && [ "${3##*/}" = "auth cert.crt" ] && [ "$4" = -k ] && [ "${5##*/}" = "auth private.key" ] && [ "$SIG_PIN" = " key password with spaces " ] || exit 44 ;;
esac
grep -q '<Nip>1111111111</Nip>' "$6"
grep -q '20250514-CR-226FB7B000-3ACF9BE4C0-10' "$6"
printf '<signed-request/>' > "${6%.xml}.signed.xml"
`
	if err := os.WriteFile(filepath.Join(dir, "godss"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}

	const authRef = "20250625-SO-2C3E6C8000-B675CF5D68-07"
	steps := []struct {
		method, path, bearer string
		status               int
		body                 string
	}{
		{"POST", "/v2/auth/challenge", "", 200, `{"challenge":"20250514-CR-226FB7B000-3ACF9BE4C0-10","timestamp":"2026-07-01T12:00:00Z","timestampMs":1782907200000,"clientIp":"127.0.0.1"}`},
		{"POST", "/v2/auth/xades-signature", "", 202, `{"referenceNumber":"` + authRef + `","authenticationToken":{"token":"pending","validUntil":"2027-01-01T00:00:00Z"}}`},
		{"GET", "/v2/auth/" + authRef, "Bearer pending", 200, `{"startDate":"2026-07-01T00:00:00Z","authenticationMethod":"QualifiedSignature","authenticationMethodInfo":{"category":"XadesSignature","code":"QualifiedSignature","displayName":"Signature"},"status":` + authStatus + `}`},
		{"POST", "/v2/auth/token/redeem", "Bearer pending", 200, `{"accessToken":{"token":"signed-session","validUntil":"2027-01-01T00:00:00Z"},"refreshToken":{"token":"refresh","validUntil":"2027-02-01T00:00:00Z"}}`},
		{"POST", "/v2/tokens/" + ref + "/approve", "Bearer signed-session", 204, ""},
		{"DELETE", "/v2/auth/sessions/current", "Bearer signed-session", 204, ""},
	}
	if wantFailure {
		steps = steps[:3] // Authentication failed before redeem and approval.
	}
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	calls := 0
	http.DefaultTransport = approvalTransport(func(r *http.Request) (*http.Response, error) {
		if calls >= len(steps) {
			return nil, fmt.Errorf("extra request %s", r.URL.Path)
		}
		s := steps[calls]
		calls++
		if r.Method != s.method || r.URL.Path != s.path || r.Header.Get("Authorization") != s.bearer {
			return nil, fmt.Errorf("unexpected method, path or credentials on request %d: %s %s", calls, r.Method, r.URL.Path)
		}
		if s.path == "/v2/auth/xades-signature" {
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "<signed-request/>" {
				return nil, fmt.Errorf("unexpected signature request")
			}
		}
		return &http.Response{StatusCode: s.status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(s.body))}, nil
	})
	parser, err := kong.New(&Cmd{}, kong.Bind(&config.Config{Env: "TEST"}))
	if err != nil {
		t.Fatal(err)
	}
	commandCtx, err := parser.Parse(args)
	if err != nil {
		t.Fatal(err)
	}
	err = commandCtx.Run()
	if os.Getenv("SIG_PIN") != "card-pin" {
		t.Fatal("command changed the parent process PIN")
	}
	if wantFailure {
		if err == nil {
			t.Fatal("expected approver authentication to fail")
		}
		output := apperrors.Format(err)
		if !strings.Contains(output, authRef) || !strings.Contains(output, "415 (Uwierzytelnianie zakończone niepowodzeniem)") ||
			strings.Count(output, "Sprawdź, czy") != 1 || !strings.Contains(output, "uprawnienia") {
			t.Fatalf("unexpected CLI error: %s", output)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	if calls != len(steps) {
		t.Fatalf("calls = %d, want %d", calls, len(steps))
	}
}

func TestSigningFailureStopsApproval(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test signer uses a POSIX shell")
	}
	bin := t.TempDir()
	work := t.TempDir()
	t.Setenv("TMPDIR", work)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(bin, "godss"), []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	calls := 0
	http.DefaultTransport = approvalTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls != 1 || r.URL.Path != "/v2/auth/challenge" {
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
		const challenge = `{"challenge":"20250514-CR-226FB7B000-3ACF9BE4C0-10","timestamp":"2026-07-01T12:00:00Z","timestampMs":1782907200000,"clientIp":"127.0.0.1"}`
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(challenge))}, nil
	})
	cmd := CardApproveCmd{ApprovalTarget: ApprovalTarget{Identifier: "1111111111", Reference: "20261106-EC-22ACCDF000-5CDA6D5392-B0"}}
	if err := cmd.Run(&config.Config{Env: "TEST"}); err == nil || !strings.Contains(err.Error(), "godss card signing failed") {
		t.Fatalf("signing error = %v", err)
	}
	files, err := os.ReadDir(work)
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary signing files after failure: %v, %v", files, err)
	}
}
