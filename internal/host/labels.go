package host

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Label key constants for sproot-managed sprites.
const (
	labelBase   = "sproot"          // marks a sprite as managed by sproot
	labelTarget = "sproot-target"   // named target used at setup/push; empty = flat phases
	labelSource = "sproot-source"   // "git" or "local"
	labelRepo   = "sproot-repo"     // git repo URL (git source only)
	labelRef    = "sproot-ref"      // git ref (git source only)
	labelSHA    = "sproot-sha"      // first 12 hex chars of sha256(sproot.yaml)
)

// ConfigMeta holds the metadata stored as sprite labels after each setup or push.
type ConfigMeta struct {
	Target string // named target; empty for flat/default
	Source string // "git" or "local"
	Repo   string // git repo URL (empty for local)
	Ref    string // git ref (empty for local)
	SHA    string // first 12 hex chars of sha256 of the applied sproot.yaml
}

// Labels returns the full set of labels to write to the sprite, including the
// base "sproot" marker. Always replaces all labels.
func (m ConfigMeta) Labels() []string {
	labels := []string{
		labelBase,
		labelTarget + "=" + m.Target,
		labelSource + "=" + m.Source,
		labelSHA + "=" + m.SHA,
	}
	if m.Repo != "" {
		labels = append(labels, labelRepo+"="+m.Repo)
	}
	if m.Ref != "" {
		labels = append(labels, labelRef+"="+m.Ref)
	}
	return labels
}

// ParseConfigMeta reads ConfigMeta from a sprite's label slice.
// Fields not present in labels are left as empty strings.
func ParseConfigMeta(labels []string) ConfigMeta {
	m := ConfigMeta{}
	for _, l := range labels {
		k, v, ok := strings.Cut(l, "=")
		if !ok {
			continue
		}
		switch k {
		case labelTarget:
			m.Target = v
		case labelSource:
			m.Source = v
		case labelRepo:
			m.Repo = v
		case labelRef:
			m.Ref = v
		case labelSHA:
			m.SHA = v
		}
	}
	return m
}

// ConfigSHA returns the first 12 hex characters of the SHA256 of the given bytes.
func ConfigSHA(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:])[:12]
}
