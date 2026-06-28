package main
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"fyne.io/fyne/v2"
)
type dropTarget struct {
	path    string
	obj     fyne.CanvasObject
	isTrash bool
}
func hitTestTargets(pos fyne.Position, targets []dropTarget, excludePath string) *dropTarget {
	driver := fyne.CurrentApp().Driver()
	for i := range targets {
		t := &targets[i]
		if t.path == excludePath {
			continue
		}
		topLeft := driver.AbsolutePositionForObject(t.obj)
		size := t.obj.Size()
		if pos.X >= topLeft.X && pos.X <= topLeft.X+size.Width &&
			pos.Y >= topLeft.Y && pos.Y <= topLeft.Y+size.Height {
			return t
		}
	}
	return nil
}
func moveFile(src, destDir string) error {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}
	if filepath.Dir(absSrc) == absDest {
		return nil
	}
	info, err := os.Lstat(absSrc)
	if err != nil {
		return err
	}
	if info.IsDir() && (absDest == absSrc || strings.HasPrefix(absDest+string(os.PathSeparator), absSrc+string(os.PathSeparator))) {
		return fmt.Errorf("cannot move a folder into itself")
	}
	destPath := filepath.Join(absDest, filepath.Base(absSrc))
	if _, err := os.Lstat(destPath); err == nil {
		return fmt.Errorf("%s already exists at destination", filepath.Base(absSrc))
	}
	if err := os.Rename(absSrc, destPath); err != nil {
		if copyErr := copyAny(absSrc, destPath); copyErr != nil {
			return copyErr
		}
		if rmErr := os.RemoveAll(absSrc); rmErr != nil {
			return rmErr
		}
	}
	return nil
}
