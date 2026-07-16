package cmd

import (
	"errors"
	"testing"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
)

func TestFriendlyErrorRewritesUnauthorized(t *testing.T) {
	err := &forgebit.APIError{StatusCode: 401, Message: "Unauthenticated."}

	got := friendlyError(err)

	want := "Your session has expired or was revoked, run `forgebit login` again"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFriendlyErrorPassesOtherErrorsThrough(t *testing.T) {
	err := errors.New("network unreachable")

	if got := friendlyError(err); got != "network unreachable" {
		t.Fatalf("expected passthrough, got %q", got)
	}
}

func TestFriendlyErrorPassesOtherStatusCodesThrough(t *testing.T) {
	err := &forgebit.APIError{StatusCode: 403, Message: "Forbidden."}

	if got := friendlyError(err); got != "Forbidden." {
		t.Fatalf("expected passthrough, got %q", got)
	}
}
