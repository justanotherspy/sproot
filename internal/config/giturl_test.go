package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNormalizeGitHubCloneURL(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		want      string
		rewritten bool
	}{
		{
			name:      "scp form with .git",
			in:        "git@github.com:justanotherspy/sprite.git",
			want:      "https://github.com/justanotherspy/sprite.git",
			rewritten: true,
		},
		{
			name:      "scp form without .git",
			in:        "git@github.com:owner/repo",
			want:      "https://github.com/owner/repo",
			rewritten: true,
		},
		{
			name:      "ssh:// form",
			in:        "ssh://git@github.com/owner/repo.git",
			want:      "https://github.com/owner/repo.git",
			rewritten: true,
		},
		{
			name:      "ssh:// form with explicit port",
			in:        "ssh://git@github.com:22/owner/repo.git",
			want:      "https://github.com/owner/repo.git",
			rewritten: true,
		},
		{
			name:      "already https unchanged",
			in:        "https://github.com/owner/repo.git",
			want:      "https://github.com/owner/repo.git",
			rewritten: false,
		},
		{
			name:      "non-github ssh unchanged",
			in:        "git@gitlab.com:owner/repo.git",
			want:      "git@gitlab.com:owner/repo.git",
			rewritten: false,
		},
		{
			name:      "local file url unchanged",
			in:        "file:///tmp/repo",
			want:      "file:///tmp/repo",
			rewritten: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rewritten := NormalizeGitHubCloneURL(tt.in)
			if got != tt.want || rewritten != tt.rewritten {
				t.Errorf("NormalizeGitHubCloneURL(%q) = (%q, %v); want (%q, %v)", tt.in, got, rewritten, tt.want, tt.rewritten)
			}
		})
	}
}

func TestGitHubHTTPSAuthArgs(t *testing.T) {
	t.Run("empty token returns nil", func(t *testing.T) {
		if got := GitHubHTTPSAuthArgs("", "https://github.com/owner/repo.git"); got != nil {
			t.Errorf("got %v; want nil", got)
		}
	})

	t.Run("non-github https returns nil", func(t *testing.T) {
		if got := GitHubHTTPSAuthArgs("tok", "https://example.com/owner/repo.git"); got != nil {
			t.Errorf("got %v; want nil", got)
		}
	})

	t.Run("ssh url returns nil", func(t *testing.T) {
		if got := GitHubHTTPSAuthArgs("tok", "git@github.com:owner/repo.git"); got != nil {
			t.Errorf("got %v; want nil (auth header only applies to https github URLs)", got)
		}
	})

	t.Run("token produces scoped basic auth header", func(t *testing.T) {
		args := GitHubHTTPSAuthArgs("s3cret", "https://github.com/owner/repo.git")
		if len(args) != 2 || args[0] != "-c" {
			t.Fatalf("got %v; want a single -c k=v pair", args)
		}
		want := "http.https://github.com/.extraheader=AUTHORIZATION: basic " +
			base64.StdEncoding.EncodeToString([]byte("x-access-token:s3cret"))
		if args[1] != want {
			t.Errorf("arg = %q; want %q", args[1], want)
		}
		// The raw token must never appear verbatim; only the base64 form.
		if strings.Contains(args[1], "s3cret") {
			t.Errorf("raw token leaked into args: %q", args[1])
		}
	})
}
