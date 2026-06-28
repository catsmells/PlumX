package main
import (
	"bytes"
	"image"
	"os"
	"github.com/dhowden/tag"
)
func loadAlbumArt(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	meta, err := tag.ReadFrom(f)
	if err != nil {
		return nil
	}
	pic := meta.Picture()
	if pic == nil || len(pic.Data) == 0 {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(pic.Data))
	if err != nil {
		return nil
	}
	return img
}
