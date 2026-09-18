// Package archiveutil creates and extracts tar.gz archives of a
// directory tree, shared by server backups (M10) and world export/import
// (M17) — both are "snapshot this directory to a single file and back".
package archiveutil

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CreateTarGz writes a gzip-compressed tar archive of every file under
// srcDir to destFile, with paths stored relative to srcDir.
func CreateTarGz(srcDir, destFile string) (err error) {
	out, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}()

	gw := gzip.NewWriter(out)
	defer func() {
		if cerr := gw.Close(); err == nil {
			err = cerr
		}
	}()

	tw := tar.NewWriter(gw)
	defer func() {
		if cerr := tw.Close(); err == nil {
			err = cerr
		}
	}()

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == srcDir {
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		// Only regular files and directories are supported; anything
		// else (symlinks, sockets, devices) is skipped rather than
		// failing the whole backup over an unusual entry.
		if !d.Type().IsRegular() && !d.IsDir() {
			return nil
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if d.IsDir() {
			hdr.Name += "/"
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}

// ExtractTarGz unpacks a gzip-compressed tar archive created by
// CreateTarGz into destDir, which must already exist. Entries that would
// extract outside destDir are rejected (zip-slip protection).
func ExtractTarGz(srcFile, destDir string) error {
	f, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("opening %s as gzip: %w", srcFile, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return err
		}

		switch {
		case hdr.FileInfo().IsDir():
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case hdr.Typeflag == tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		default:
			// Skip anything that isn't a plain file or directory.
		}
	}
}

// safeJoin joins base and name, rejecting any name that would escape
// base once cleaned (a zip-slip/path-traversal guard for archive
// extraction).
func safeJoin(base, name string) (string, error) {
	target := filepath.Join(base, name)
	baseWithSep := filepath.Clean(base) + string(os.PathSeparator)
	if target != filepath.Clean(base) && !strings.HasPrefix(target, baseWithSep) {
		return "", fmt.Errorf("archive entry %q escapes destination directory", name)
	}
	return target, nil
}
