package host

import (
	"strings"
	"testing"
)

func TestConfigSHA(t *testing.T) {
	sha := ConfigSHA([]byte("hello"))
	if len(sha) != 12 {
		t.Errorf("SHA length: got %d, want 12", len(sha))
	}
	// SHA is deterministic.
	if ConfigSHA([]byte("hello")) != sha {
		t.Error("SHA is not deterministic")
	}
	// Different input gives different SHA.
	if ConfigSHA([]byte("world")) == sha {
		t.Error("different input produced same SHA")
	}
}

func TestConfigMeta_Labels_Git(t *testing.T) {
	m := ConfigMeta{
		Target: "web",
		Source: "git",
		Repo:   "git@github.com:user/repo.git",
		Ref:    "main",
		SHA:    "abc123def456",
	}
	labels := m.Labels()

	hasLabel := func(s string) bool {
		for _, l := range labels {
			if l == s {
				return true
			}
		}
		return false
	}
	mustHaveKV := func(key, val string) {
		t.Helper()
		want := key + "=" + val
		if !hasLabel(want) {
			t.Errorf("labels missing %q; got %v", want, labels)
		}
	}

	if !hasLabel(labelBase) {
		t.Errorf("labels missing %q; got %v", labelBase, labels)
	}
	mustHaveKV(labelTarget, "web")
	mustHaveKV(labelSource, "git")
	mustHaveKV(labelRepo, "git@github.com:user/repo.git")
	mustHaveKV(labelRef, "main")
	mustHaveKV(labelSHA, "abc123def456")
}

func TestConfigMeta_Labels_Local(t *testing.T) {
	m := ConfigMeta{
		Source: "local",
		Repo:   "/home/user/config",
		SHA:    "abc123def456",
	}
	labels := m.Labels()
	// Ref and Target should not appear for local source with no target.
	for _, l := range labels {
		if strings.HasPrefix(l, labelRef+"=") {
			t.Errorf("unexpected %s label for local source: %v", labelRef, labels)
		}
		if strings.HasPrefix(l, labelTarget+"=") {
			t.Errorf("unexpected %s label when Target is empty: %v", labelTarget, labels)
		}
	}
}

func TestConfigMeta_Labels_EmptyTarget(t *testing.T) {
	m := ConfigMeta{
		Target: "",
		Source: "git",
		Repo:   "git@github.com:user/repo.git",
		Ref:    "main",
		SHA:    "abc123def456",
	}
	labels := m.Labels()
	for _, l := range labels {
		if strings.HasPrefix(l, labelTarget+"=") {
			t.Errorf("sproot-target label must be omitted when Target is empty, got: %v", labels)
		}
	}
}

func TestParseConfigMeta(t *testing.T) {
	labels := []string{
		"sproot",
		"sproot-target=web",
		"sproot-source=git",
		"sproot-repo=git@github.com:user/repo.git",
		"sproot-ref=main",
		"sproot-sha=abc123def456",
	}
	m := ParseConfigMeta(labels)
	if m.Target != "web" {
		t.Errorf("Target: got %q, want web", m.Target)
	}
	if m.Source != "git" {
		t.Errorf("Source: got %q, want git", m.Source)
	}
	if m.SHA != "abc123def456" {
		t.Errorf("SHA: got %q, want abc123def456", m.SHA)
	}
}

func TestParseConfigMeta_Empty(t *testing.T) {
	m := ParseConfigMeta([]string{"sproot"})
	if m.SHA != "" || m.Target != "" || m.Source != "" {
		t.Errorf("expected empty meta, got %+v", m)
	}
}

func TestConfigMeta_RoundTrip(t *testing.T) {
	original := ConfigMeta{
		Target: "ml",
		Source: "git",
		Repo:   "git@github.com:u/r.git",
		Ref:    "main",
		SHA:    "deadbeef1234",
	}
	parsed := ParseConfigMeta(original.Labels())
	if parsed != original {
		t.Errorf("round trip failed: got %+v, want %+v", parsed, original)
	}
}
