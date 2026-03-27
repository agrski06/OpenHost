package install

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/openhost/runner/internal/system"
)

// HTTPDownloadInstall installs a game server by downloading a file over HTTP.
type HTTPDownloadInstall struct {
	URL          string
	DestFilename string
	ExtractZip   bool
	ExtractTar   bool
}

// Install downloads the file and optionally extracts it.
func (h *HTTPDownloadInstall) Install(_ context.Context, _, serverRoot string) error {
	if err := os.MkdirAll(serverRoot, 0755); err != nil {
		return fmt.Errorf("create server root %s: %w", serverRoot, err)
	}

	destPath := filepath.Join(serverRoot, h.DestFilename)

	if err := system.DownloadWithRetry(h.URL, destPath, 5); err != nil {
		return fmt.Errorf("download %s: %w", h.URL, err)
	}

	if h.ExtractZip {
		if err := extractZip(destPath, serverRoot); err != nil {
			return fmt.Errorf("extract zip %s: %w", destPath, err)
		}
	}

	if h.ExtractTar {
		if err := extractTarGz(destPath, serverRoot); err != nil {
			return fmt.Errorf("extract tar %s: %w", destPath, err)
		}
	}

	return nil
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
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

func extractTarGz(tarPath, destDir string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		name := filepath.ToSlash(header.Name)
		if strings.Contains(name, "..") {
			return fmt.Errorf("tar entry with path traversal: %s", header.Name)
		}

		target := filepath.Join(destDir, name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			_ = out.Close()
			if header.FileInfo().Mode()&0111 != 0 {
				_ = os.Chmod(target, 0755)
			}
		}
	}

	return nil
}
