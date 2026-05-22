package tty

import (
	"bytes"
	"testing"
)

func TestIsColor_NonFileWriterIsPlain(t *testing.T) {
	if IsColor(&bytes.Buffer{}) {
		t.Error("a bytes.Buffer is not a terminal; IsColor should be false")
	}
}

func TestIsColor_NoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if IsColor(&bytes.Buffer{}) {
		t.Error("NO_COLOR set: IsColor should be false")
	}
}

func TestWidth_NonFileWriterIsZero(t *testing.T) {
	if got := Width(&bytes.Buffer{}); got != 0 {
		t.Errorf("Width(buffer) = %d, want 0", got)
	}
}
