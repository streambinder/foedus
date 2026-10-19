package handlers

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/streambinder/foedus/internal/database"
)

var (
	checkerWhite = color.NRGBA{R: 254, G: 254, B: 254, A: 255}
	checkerGrey  = color.NRGBA{R: 226, G: 226, B: 226, A: 255}
	flowerBeige  = color.NRGBA{R: 216, G: 196, B: 170, A: 255}
)

// checkerboardImage draws a two-tone checkerboard with a solid
// chromatic blob in the middle, the shape of a stock "transparent"
// corner graphic with the grid baked in.
func checkerboardImage(width, height, cell int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x/cell+y/cell)%2 == 0 {
				img.SetNRGBA(x, y, checkerWhite)
			} else {
				img.SetNRGBA(x, y, checkerGrey)
			}
		}
	}
	for y := height/2 - 15; y < height/2+15; y++ {
		for x := width/2 - 15; x < width/2+15; x++ {
			img.SetNRGBA(x, y, flowerBeige)
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func decodePNG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func alphaAt(img image.Image, x, y int) uint32 {
	_, _, _, a := img.At(x, y).RGBA()
	return a
}

func TestStripFakeTransparencyCheckerboard(t *testing.T) {
	data := encodePNG(t, checkerboardImage(160, 120, 16))
	mime, out, ok := stripFakeTransparency("image/png", data)
	if !ok {
		t.Fatal("checkerboard not detected")
	}
	if mime != "image/png" {
		t.Errorf("mime = %q", mime)
	}
	img := decodePNG(t, out)
	if a := alphaAt(img, 2, 2); a != 0 {
		t.Errorf("background pixel alpha = %d, want 0", a)
	}
	if a := alphaAt(img, 80, 60); a != 0xffff {
		t.Errorf("motif pixel alpha = %d, want opaque", a)
	}
	r, g, b, _ := img.At(80, 60).RGBA()
	want := color.NRGBA{R: 216, G: 196, B: 170, A: 255}
	wr, wg, wb, _ := want.RGBA()
	if r != wr || g != wg || b != wb {
		t.Errorf("motif colour = (%d, %d, %d), want (%d, %d, %d)", r, g, b, wr, wg, wb)
	}
}

func TestStripFakeTransparencyCheckerboardJPEG(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, checkerboardImage(160, 120, 16), &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	_, out, ok := stripFakeTransparency("image/jpeg", buf.Bytes())
	if !ok {
		t.Fatal("jpeg checkerboard not detected")
	}
	img := decodePNG(t, out)
	if a := alphaAt(img, 2, 2); a != 0 {
		t.Errorf("background pixel alpha = %d, want 0", a)
	}
	if a := alphaAt(img, 80, 60); a != 0xffff {
		t.Errorf("motif pixel alpha = %d, want opaque", a)
	}
}

func TestStripFakeTransparencyKeepsPlainBackground(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 160, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 160; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	for y := 45; y < 75; y++ {
		for x := 65; x < 95; x++ {
			img.SetNRGBA(x, y, flowerBeige)
		}
	}
	data := encodePNG(t, img)
	if _, out, ok := stripFakeTransparency("image/png", data); ok || !bytes.Equal(out, data) {
		t.Error("plain white background must be left untouched")
	}
}

func TestStripFakeTransparencyKeepsRealTransparency(t *testing.T) {
	img := checkerboardImage(160, 120, 16)
	for y := 0; y < 120; y++ {
		for x := 0; x < 160; x++ {
			if img.NRGBAAt(x, y) != flowerBeige {
				c := img.NRGBAAt(x, y)
				c.A = 0
				img.SetNRGBA(x, y, c)
			}
		}
	}
	data := encodePNG(t, img)
	if _, out, ok := stripFakeTransparency("image/png", data); ok || !bytes.Equal(out, data) {
		t.Error("image with real transparency must be left untouched")
	}
}

func TestStripFakeTransparencyRejects(t *testing.T) {
	checker := encodePNG(t, checkerboardImage(160, 120, 16))
	if _, _, ok := stripFakeTransparency("image/webp", checker); ok {
		t.Error("webp input must not be processed")
	}
	if _, _, ok := stripFakeTransparency("image/png", []byte("not an image")); ok {
		t.Error("undecodable input must not be processed")
	}
	tiny := encodePNG(t, checkerboardImage(16, 16, 4))
	if _, _, ok := stripFakeTransparency("image/png", tiny); ok {
		t.Error("tiny image must not be processed")
	}
	// vertical stripes are all border-connected background: the cleared
	// share gate rejects them instead of emptying the whole image
	stripes := image.NewNRGBA(image.Rect(0, 0, 160, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 160; x++ {
			if (x/16)%2 == 0 {
				stripes.SetNRGBA(x, y, checkerWhite)
			} else {
				stripes.SetNRGBA(x, y, checkerGrey)
			}
		}
	}
	if _, _, ok := stripFakeTransparency("image/png", encodePNG(t, stripes)); ok {
		t.Error("full-bleed stripes must not be stripped")
	}
}

func TestResolveFloraImageMediaID(t *testing.T) {
	setupDB(t)
	q := database.DB

	checker := encodePNG(t, checkerboardImage(160, 120, 16))
	raw := "data:image/png;base64," + base64.StdEncoding.EncodeToString(checker)
	id, err := resolveFloraImageMediaID(q, raw, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	media, err := database.GetMedia(id)
	if err != nil {
		t.Fatal(err)
	}
	if media.Mime != "image/png" {
		t.Errorf("stored mime = %q", media.Mime)
	}
	img := decodePNG(t, media.Bytes)
	if a := alphaAt(img, 2, 2); a != 0 {
		t.Errorf("stored background alpha = %d, want 0", a)
	}
	if a := alphaAt(img, 80, 60); a != 0xffff {
		t.Errorf("stored motif alpha = %d, want opaque", a)
	}

	// an image without a fake grid is stored byte for byte
	plain := encodePNG(t, image.NewNRGBA(image.Rect(0, 0, 40, 40)))
	raw = "data:image/png;base64," + base64.StdEncoding.EncodeToString(plain)
	id, err = resolveFloraImageMediaID(q, raw, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	media, err = database.GetMedia(id)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(media.Bytes, plain) {
		t.Error("plain image must be stored unchanged")
	}
}
