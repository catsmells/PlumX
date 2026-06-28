package main
import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"
)
func trashDirs() (filesDir, infoDir string, err error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	trash := filepath.Join(dataHome, "Trash")
	filesDir = filepath.Join(trash, "files")
	infoDir = filepath.Join(trash, "info")
	if err := os.MkdirAll(filesDir, 0700); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(infoDir, 0700); err != nil {
		return "", "", err
	}
	return filesDir, infoDir, nil
}
func trashFilesDir() (string, error) {
	filesDir, _, err := trashDirs()
	return filesDir, err
}
func uniqueTrashName(filesDir, infoDir, base string) string {
	name := base
	for i := 1; ; i++ {
		_, errF := os.Lstat(filepath.Join(filesDir, name))
		_, errI := os.Lstat(filepath.Join(infoDir, name+".trashinfo"))
		if errF != nil && errI != nil {
			return name
		}
		ext := filepath.Ext(base)
		name = fmt.Sprintf("%s_%d%s", base[:len(base)-len(ext)], i, ext)
	}
}
func moveToTrash(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	filesDir, infoDir, err := trashDirs()
	if err != nil {
		return err
	}
	name := uniqueTrashName(filesDir, infoDir, filepath.Base(absPath))
	destPath := filepath.Join(filesDir, name)
	infoPath := filepath.Join(infoDir, name+".trashinfo")
	info := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n",
		(&url.URL{Path: absPath}).EscapedPath(),
		time.Now().Format("2006-01-02T15:04:05"))
	if err := os.WriteFile(infoPath, []byte(info), 0600); err != nil {
		return err
	}
	if err := os.Rename(absPath, destPath); err != nil {
		if copyErr := copyAny(absPath, destPath); copyErr != nil {
			os.Remove(infoPath)
			return copyErr
		}
		if rmErr := os.RemoveAll(absPath); rmErr != nil {
			os.Remove(infoPath)
			os.RemoveAll(destPath)
			return rmErr
		}
	}
	return nil
}
func emptyTrash() error {
	filesDir, infoDir, err := trashDirs()
	if err != nil {
		return err
	}
	for _, dir := range []string{filesDir, infoDir} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
func copyAny(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return copyFile(src, dst)
	}
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyAny(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
