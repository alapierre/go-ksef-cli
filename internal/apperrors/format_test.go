package apperrors

import (
	"strings"
	"testing"

	"github.com/alapierre/go-ksef-client/ksef"
)

func TestFormatKSeFAPIError(t *testing.T) {
	err := &ksef.ApiError{
		Status:  400,
		Message: "Błąd walidacji danych wejściowych.",
		Details: []ksef.ErrorDetail{
			{
				Code:    21405,
				Message: "Błąd walidacji danych wejściowych.",
				Details: []string{
					"'dateRange.to' must be greater than or equal to 'dateRange.from'.",
				},
			},
		},
	}

	got := Format(err)

	for _, want := range []string{
		"KSeF returned an error.",
		"HTTP status: 400",
		"Message: Błąd walidacji danych wejściowych.",
		"Code 21405: Błąd walidacji danych wejściowych.",
		"- 'dateRange.to' must be greater than or equal to 'dateRange.from'.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatted error does not contain %q:\n%s", want, got)
		}
	}
}

func TestFormatNonKSeFError(t *testing.T) {
	got := Format(assertErr("plain error"))
	if got != "error: plain error" {
		t.Fatalf("Format() = %q, want %q", got, "error: plain error")
	}
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}
