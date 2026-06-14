package apperrors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/alapierre/go-ksef-client/ksef"
)

func Format(err error) string {
	var apiErr *ksef.ApiError
	if errors.As(err, &apiErr) {
		return formatKSeFAPIError(apiErr)
	}
	return fmt.Sprintf("error: %v", err)
}

func formatKSeFAPIError(err *ksef.ApiError) string {
	var b strings.Builder
	b.WriteString("KSeF returned an error.\n")

	if err.Status > 0 {
		fmt.Fprintf(&b, "HTTP status: %d\n", err.Status)
	}
	if msg := strings.TrimSpace(err.Message); msg != "" {
		fmt.Fprintf(&b, "Message: %s\n", msg)
	}

	if len(err.Details) == 0 {
		return strings.TrimRight(b.String(), "\n")
	}

	b.WriteString("Details:\n")
	for i, detail := range err.Details {
		fmt.Fprintf(&b, "  %d. ", i+1)
		if detail.Code != 0 {
			fmt.Fprintf(&b, "Code %d: ", detail.Code)
		}
		if detail.Message != "" {
			b.WriteString(detail.Message)
		} else {
			b.WriteString("KSeF API error")
		}
		b.WriteByte('\n')

		for _, text := range detail.Details {
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			fmt.Fprintf(&b, "     - %s\n", text)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}
