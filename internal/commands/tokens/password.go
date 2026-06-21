package tokens

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func resolvePassword(provided string, confirm bool) (string, error) {
	provided = strings.TrimSpace(provided)
	if provided != "" {
		return provided, nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("backup password not provided; use --password or KSEF_BACKUP_PASSWORD")
	}

	fmt.Print("Backup password: ")
	p1, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read backup password: %w", err)
	}

	password := strings.TrimSpace(string(p1))
	if password == "" {
		return "", fmt.Errorf("backup password cannot be empty")
	}

	if !confirm {
		return password, nil
	}

	fmt.Print("Confirm backup password: ")
	p2, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read backup password confirmation: %w", err)
	}

	if password != strings.TrimSpace(string(p2)) {
		return "", fmt.Errorf("backup password confirmation does not match")
	}

	return password, nil
}
