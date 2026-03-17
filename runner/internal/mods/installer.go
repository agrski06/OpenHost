package mods

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func SanitizeZipEntryPath(name string) (string, error) {
	clean := strings.ReplaceAll(name, "\\", "/")
	clean = path.Clean(clean)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "." || clean == "" {
		return "", fmt.Errorf("empty archive entry path")
	}
	if strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("path traversal detected for %q", name)
	}
	return filepath.FromSlash(clean), nil
}

func ExtractZipToDir(archive []byte, destination string) error {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return err
	}

	destination = filepath.Clean(destination)

	for _, file := range reader.File {
		cleanPath, err := SanitizeZipEntryPath(file.Name)
		if err != nil {
			continue
		}
		targetPath := filepath.Clean(filepath.Join(destination, cleanPath))
		if !strings.HasPrefix(targetPath, destination+string(os.PathSeparator)) && targetPath != destination {
			return fmt.Errorf("archive entry escaped destination: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			_ = rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			_ = out.Close()
			_ = rc.Close()
			return err
		}
		if err := out.Close(); err != nil {
			_ = rc.Close()
			return err
		}
		if err := rc.Close(); err != nil {
			return err
		}
	}

	return nil
}

func InstallArchive(archive []byte, destination string) error {
	tempDir, err := os.MkdirTemp("", "openhost-mod-install-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	if err := ExtractZipToDir(archive, tempDir); err != nil {
		return err
	}

	root, err := resolveInstallRoot(tempDir)
	if err != nil {
		return err
	}
	return CopyTree(root, destination)
}

func resolveInstallRoot(base string) (string, error) {
	current := base
	for {
		entries, err := os.ReadDir(current)
		if err != nil {
			return "", err
		}
		if len(entries) != 1 || !entries[0].IsDir() {
			return current, nil
		}
		current = filepath.Join(current, entries[0].Name())
	}
}

func CopyTree(source string, destination string) error {
	return filepath.WalkDir(source, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(current)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			_ = in.Close()
			return err
		}
		_, err = io.Copy(out, in)
		closeOutErr := out.Close()
		closeInErr := in.Close()
		if err != nil {
			return err
		}
		if closeOutErr != nil {
			return closeOutErr
		}
		return closeInErr
	})
}
