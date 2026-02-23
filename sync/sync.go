package sync

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Options configures a sync operation.
type Options struct {
	Src    string      // source directory
	Dst    Destination // destination
	DryRun bool        // if true, print actions without making changes
	Delete bool        // if true, remove destination objects absent from Src
}

// Sync copies files from opts.Src to opts.Dst, skipping files that are
// already up to date (matched by size and modification time).
func Sync(ctx context.Context, opts Options) error {
	if err := validateSrc(opts.Src); err != nil {
		return err
	}
	if err := syncFiles(ctx, opts); err != nil {
		return err
	}
	if opts.Delete {
		return deleteExtras(ctx, opts)
	}
	return nil
}

func syncFiles(ctx context.Context, opts Options) error {
	return filepath.WalkDir(opts.Src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		rel, err := filepath.Rel(opts.Src, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel) // S3 keys use forward slashes

		info, err := d.Info()
		if err != nil {
			return err
		}

		meta, err := opts.Dst.Stat(ctx, rel)
		if err != nil {
			return fmt.Errorf("stat %s: %w", rel, err)
		}
		if meta != nil && meta.ModTime.Equal(info.ModTime().Truncate(1e9)) && meta.Size == info.Size() {
			return nil // already up to date
		}

		fmt.Printf("upload %s\n", rel)
		if opts.DryRun {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		return opts.Dst.Put(ctx, rel, f, info.Size(), info.ModTime())
	})
}

func deleteExtras(ctx context.Context, opts Options) error {
	keys, err := opts.Dst.List(ctx)
	if err != nil {
		return err
	}

	for _, key := range keys {
		localPath := filepath.Join(opts.Src, filepath.FromSlash(key))
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			fmt.Printf("delete %s\n", key)
			if !opts.DryRun {
				if err := opts.Dst.Delete(ctx, key); err != nil {
					return fmt.Errorf("delete %s: %w", key, err)
				}
			}
		}
	}
	return nil
}

func validateSrc(src string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source %q is not a directory", src)
	}
	return nil
}
