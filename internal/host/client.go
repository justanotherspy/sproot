package host

import (
	"context"
	"io"
	"io/fs"

	sprites "github.com/superfly/sprites-go"
)

// SpriteHandle abstracts per-sprite operations, enabling test mocking.
type SpriteHandle interface {
	WriteFile(path string, data []byte, perm fs.FileMode) error
	ReadFile(path string) ([]byte, error)
	RunCommand(name string, args, env []string, stdout, stderr io.Writer) error
}

// SpritesClient abstracts sprite lifecycle operations, enabling test mocking.
type SpritesClient interface {
	CreateSprite(ctx context.Context, name string, cfg *sprites.SpriteConfig) (SpriteHandle, error)
	GetHandle(name string) SpriteHandle
	DestroySprite(ctx context.Context, name string) error
}

// NewClient constructs a real SpritesClient from an API token.
func NewClient(token string) SpritesClient {
	return &realClient{c: sprites.New(token)}
}

type realClient struct{ c *sprites.Client }

func (r *realClient) CreateSprite(ctx context.Context, name string, cfg *sprites.SpriteConfig) (SpriteHandle, error) {
	s, err := r.c.CreateSprite(ctx, name, cfg)
	if err != nil {
		return nil, err
	}
	return &realHandle{s: s}, nil
}

func (r *realClient) GetHandle(name string) SpriteHandle {
	return &realHandle{s: r.c.Sprite(name)}
}

func (r *realClient) DestroySprite(ctx context.Context, name string) error {
	return r.c.DestroySprite(ctx, name)
}

type realHandle struct{ s *sprites.Sprite }

func (h *realHandle) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return h.s.Filesystem().WriteFile(path, data, perm)
}

func (h *realHandle) ReadFile(path string) ([]byte, error) {
	return h.s.Filesystem().ReadFile(path)
}

func (h *realHandle) RunCommand(name string, args, env []string, stdout, stderr io.Writer) error {
	cmd := h.s.Command(name, args...)
	cmd.Env = env
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
