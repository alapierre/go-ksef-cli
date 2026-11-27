package tokens

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/alapierre/go-ksef-client/ksef/tpl"
)

func (c *ApprovalTarget) authenticate(ctx context.Context, auth *ksef.AuthFacade, signerArgs []string, password string) (*api.AuthenticationTokensResponse, error) {
	helper, err := exec.LookPath("godss")
	if err != nil {
		return nil, fmt.Errorf("XAdES signing requires godss on PATH: %w", err)
	}
	challenge, err := auth.GetChallenge(ctx)
	if err != nil {
		return nil, err
	}
	request, err := tpl.RenderAuthRequestXML(tpl.AuthRequestData{
		Challenge: string(challenge.Challenge), Identifier: c.Identifier,
		SubjectIdentifierType: "certificateSubject",
	})
	if err != nil {
		return nil, err
	}
	signed, err := signWithGodss(ctx, helper, signerArgs, password, request)
	if err != nil {
		return nil, err
	}
	init, err := auth.AuthWithXades(ctx, signed)
	if err != nil {
		return nil, err
	}
	pollCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	tokens, err := auth.AuthWaitAndRedeem(pollCtx, init, time.Second)
	if err != nil {
		// SDK v1.1.0 exposes terminal authentication statuses only as plain text.
		// Match its single-line 415 error before adding command-specific guidance.
		message := err.Error()
		if strings.HasPrefix(message, "uwierzytelnianie zakończone kodem 415 (") && !strings.Contains(message, "\n") {
			return nil, fmt.Errorf("authentication reference %s: %w\nSprawdź, czy na wybranym środowisku KSeF osoba podpisująca ma uprawnienia do działania w imieniu firmy o NIP-ie podanym w -i.", init.ReferenceNumber, err)
		}
		return nil, fmt.Errorf("authentication reference %s: %w", init.ReferenceNumber, err)
	}
	return tokens, nil
}

func signWithGodss(ctx context.Context, helper string, signerArgs []string, password string, request []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "ksef-xades-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	input := filepath.Join(dir, "auth.xml")
	if err := os.WriteFile(input, request, 0600); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, helper, append(signerArgs, input)...)
	if password != "" {
		cmd.Env = append(cmd.Environ(), "SIG_PIN="+password)
	}
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("godss %s signing failed: %w", signerArgs[0], err)
	}
	return os.ReadFile(filepath.Join(dir, "auth.signed.xml"))
}
