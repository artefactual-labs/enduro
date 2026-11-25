package activities

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	bagit "github.com/artefactual-labs/bagit-gython"

	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/validation"
)

// ValidateTransferActivity validates staged transfer content before ingest.
type ValidateTransferActivity struct {
	bags *bagit.Validator
}

func NewValidateTransferActivity(bags *bagit.Validator) *ValidateTransferActivity {
	return &ValidateTransferActivity{bags: bags}
}

type ValidateTransferActivityParams struct {
	Config validation.Config
	Path   string

	CheckBagCompleteness bool
	RequireBag           bool
}

func (a *ValidateTransferActivity) Execute(ctx context.Context, params *ValidateTransferActivityParams) error {
	if params.CheckBagCompleteness {
		if _, err := checkStagedBag(ctx, params.Path, params.RequireBag, a.bags.CheckCompletenessContext); err != nil {
			return err
		}
	}
	return validation.ValidateTransfer(params.Config, params.Path)
}

// checkStagedBag returns true only for a bag that passed check. Opening the
// staged directory distinguishes a missing transfer from an ordinary directory
// with no BagIt declaration.
func checkStagedBag(ctx context.Context, path string, required bool, check func(context.Context, string) error) (bool, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return false, fmt.Errorf("open staged transfer: %w", err)
	}
	defer root.Close()

	const marker = "bagit.txt"
	info, err := root.Lstat(marker)
	if errors.Is(err, os.ErrNotExist) {
		if required {
			return false, temporal.NewNonRetryableError(errors.New("expected an unzipped BagIt bag: bagit.txt is missing at the transfer root"))
		}
		return false, nil
	}
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		info, err = resolveTransferLink(root, marker)
		if err != nil {
			return false, err
		}
	}
	if err != nil {
		return false, fmt.Errorf("inspect staged BagIt declaration: %w", err)
	}
	if !info.Mode().IsRegular() {
		return false, temporal.NewNonRetryableError(errors.New("invalid BagIt declaration: bagit.txt must be a regular file"))
	}

	// The validator opens paths by name, including manifests and data/.
	// Walk without following links and check their targets through the root.
	if err := checkTransferLinks(ctx, root); err != nil {
		return false, err
	}

	if err := check(ctx, path); err != nil {
		if errors.Is(err, bagit.ErrInvalid) {
			return false, temporal.NewNonRetryableError(fmt.Errorf("BagIt validation failed: %w", err))
		}
		return false, fmt.Errorf("could not validate staged bag: %w", err)
	}
	return true, nil
}

func checkTransferLinks(ctx context.Context, root *os.Root) error {
	return fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		mode := entry.Type()
		if mode&fs.ModeSymlink != 0 {
			info, err := resolveTransferLink(root, name)
			if err != nil {
				return err
			}
			mode = info.Mode()
		}
		if !mode.IsRegular() && !mode.IsDir() {
			return temporal.NewNonRetryableError(fmt.Errorf("unsupported file type in transfer: %q", name))
		}
		return nil
	})
}

func resolveTransferLink(root *os.Root, name string) (fs.FileInfo, error) {
	info, err := root.Stat(name)
	if err == nil {
		return info, nil
	}
	// Root rejects escaping links before the filesystem is called.
	// Preserve retries for filesystem I/O errors.
	var errno syscall.Errno
	if !errors.As(err, &errno) || errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ELOOP) || errors.Is(err, syscall.ENOTDIR) {
		return nil, temporal.NewNonRetryableError(fmt.Errorf("link %q must resolve inside the transfer: %w", name, err))
	}
	return nil, err
}
