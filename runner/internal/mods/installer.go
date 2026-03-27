package mods

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/openhost/runner/internal/core"
	"github.com/openhost/runner/internal/system"
)

// DownloadPackage downloads a mod package to destDir and returns a DownloadedMod.
func DownloadPackage(pkg core.Package, destDir string) (*core.DownloadedMod, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create dir %s: %w", destDir, err)
	}

	destPath := filepath.Join(destDir, pkg.Identifier+".zip")
	if err := system.DownloadWithRetry(pkg.URL, destPath, 3); err != nil {
		return nil, fmt.Errorf("download %s: %w", pkg.Identifier, err)
	}

	return &core.DownloadedMod{
		Identifier: pkg.Identifier,
		LocalPath:  destPath,
	}, nil
}

// ExtractZip extracts a zip archive to destDir with path traversal protection.
func ExtractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip %s: %w", zipPath, err)
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		if strings.Contains(name, "..") {
			return fmt.Errorf("zip entry with path traversal: %s", f.Name)
		}

		target := filepath.Join(destDir, name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(target)
		if err != nil {
			_ = rc.Close()
			return err
		}

		if _, err := io.Copy(out, rc); err != nil {
			_ = rc.Close()
			_ = out.Close()
			return err
		}

		_ = rc.Close()
		_ = out.Close()
	}

	return nil
}
