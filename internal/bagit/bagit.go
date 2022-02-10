package bagit

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime/debug"

	go_bagit "github.com/nyudlts/go-bagit"
)

var ErrInfoNotAvailable = errors.New("bag tool infonformation not available")

// bagTool can check the completeness and validness of bags as per the spec.
// See https://datatracker.ietf.org/doc/html/rfc8493#section-3 for more.
type bagTool interface {
	Complete(ctx context.Context, path string) error
	Valid(ctx context.Context, path string) error
	Info(ctx context.Context) (string, error)
}

var (
	bagitGo         = goTool{}
	bagitPy         = pyTool{}
	tool    bagTool = bagitGo
)

func UseGoBagit() {
	tool = bagitGo
}

func UsePyBagit() {
	tool = bagitPy
}

func Complete(ctx context.Context, path string) error {
	return tool.Complete(ctx, path)
}

func Valid(ctx context.Context, path string) error {
	return tool.Valid(ctx, path)
}

func Info(ctx context.Context) (string, error) {
	return tool.Info(ctx)
}

type pyTool struct{}

var _ bagTool = pyTool{}

func (t pyTool) cmd(ctx context.Context, args ...string) *exec.Cmd {
	const executable = "bagit.py"
	return exec.CommandContext(ctx, executable, args...)
}

func (t pyTool) Complete(ctx context.Context, path string) error {
	cmd := t.cmd(ctx, "--validate", "--completeness-only", path)
	return cmd.Run()
}

func (t pyTool) Valid(ctx context.Context, path string) error {
	cmd := t.cmd(ctx, "--validate", path)
	return cmd.Run()
}

func (t pyTool) Info(ctx context.Context) (string, error) {
	cmd := t.cmd(ctx, "--version")
	blob, err := cmd.Output()
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewReader(blob))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		return scanner.Text(), nil
	}
	return "", ErrInfoNotAvailable
}

type goTool struct{}

var _ bagTool = goTool{}

func (t goTool) Complete(ctx context.Context, path string) error {
	return go_bagit.ValidateBag(path, false, true)
}

func (t goTool) Valid(ctx context.Context, path string) error {
	return go_bagit.ValidateBag(path, false, false)
}

func (t goTool) Info(ctx context.Context) (string, error) {
	const modulePath = "github.com/nyudlts/go-bagit"
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", ErrInfoNotAvailable
	}
	for _, dep := range bi.Deps {
		if dep.Path == modulePath {
			return fmt.Sprintf("go-bagit %s (sum %s)", dep.Version, dep.Sum), nil
		}
	}
	return "", ErrInfoNotAvailable
}
