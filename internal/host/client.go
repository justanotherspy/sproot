package host

import (
	"context"
	"io"
	"io/fs"
	"os"

	sprites "github.com/superfly/sprites-go"
	"golang.org/x/term"
)

// SpriteListEntry holds the subset of sprite info needed for list output.
type SpriteListEntry struct {
	Name   string
	Status string
	Labels []string
}

// SpriteHandle abstracts per-sprite operations, enabling test mocking.
type SpriteHandle interface {
	WriteFile(path string, data []byte, perm fs.FileMode) error
	ReadFile(path string) ([]byte, error)
	RunCommand(name string, args, env []string, stdout, stderr io.Writer) error
	// Console opens an interactive TTY shell on the sprite.
	Console(env []string) error
}

// SpritesClient abstracts sprite lifecycle operations, enabling test mocking.
type SpritesClient interface {
	CreateSprite(ctx context.Context, name string, cfg *sprites.SpriteConfig, labels []string) (SpriteHandle, error)
	GetHandle(name string) SpriteHandle
	DestroySprite(ctx context.Context, name string) error
	ListSprites(ctx context.Context) ([]SpriteListEntry, error)
}

// NewClient constructs a real SpritesClient from an API token.
func NewClient(token string) SpritesClient {
	return &realClient{c: sprites.New(token)}
}

type realClient struct{ c *sprites.Client }

func (r *realClient) CreateSprite(ctx context.Context, name string, cfg *sprites.SpriteConfig, labels []string) (SpriteHandle, error) {
	s, err := r.c.CreateSpriteWithOrg(ctx, name, cfg, nil, labels)
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

func (r *realClient) ListSprites(ctx context.Context) ([]SpriteListEntry, error) {
	var entries []SpriteListEntry
	var contToken string
	for {
		page, err := r.c.ListSprites(ctx, &sprites.ListOptions{
			MaxResults:        100,
			ContinuationToken: contToken,
		})
		if err != nil {
			return nil, err
		}
		for _, info := range page.Sprites {
			entries = append(entries, SpriteListEntry{
				Name:   info.Name,
				Status: info.Status,
				Labels: info.Labels,
			})
		}
		if !page.HasMore || page.NextContinuationToken == "" {
			break
		}
		contToken = page.NextContinuationToken
	}
	return entries, nil
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

func (h *realHandle) Console(env []string) error {
	cmd := h.s.Command("bash")
	cmd.SetTTY(true)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if w, ht, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		_ = cmd.SetTTYSize(uint16(ht), uint16(w))
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err == nil {
		defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()
	}
	return cmd.Run()
}
