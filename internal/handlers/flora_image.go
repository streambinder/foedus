package handlers

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/draw"
	"image/png"
	"strings"

	_ "image/jpeg" // register the JPEG decoder for image.Decode

	"github.com/streambinder/foedus/internal/database"
)

// Uploaded corner graphics often come from stock sites whose
// "transparent" downloads carry a baked-in checkerboard instead of
// real transparency. resolveFloraImageMediaID strips that pattern at
// upload time, so the stored media holds true transparency and the
// public rendering blends cleanly into the page.
func resolveFloraImageMediaID(q database.Querier, rawImage, rawMediaID string, existingMediaID int) (int, error) {
	if image := strings.TrimSpace(rawImage); image != "" {
		if mime, data, err := decodeDataURI(image); err == nil {
			if newMime, stripped, ok := stripFakeTransparency(mime, data); ok {
				rawImage = "data:" + newMime + ";base64," + base64.StdEncoding.EncodeToString(stripped)
			}
		}
	}
	return resolveImageMediaID(q, rawImage, rawMediaID, existingMediaID, true)
}

const (
	// the two checkerboard tones together must cover this share of the
	// image before the pattern test runs at all
	checkerMinCoverage = 0.45
	// per-channel distance from a tone centroid for a pixel to count
	// as part of that tone
	checkerToneTolerance = 28
	// a checkerboard pixel must be near-neutral: this keeps pale but
	// coloured motif pixels (sage, beige) out of the cleared set
	checkerMaxSpread = 14
)

// stripFakeTransparency detects a baked-in transparency checkerboard
// and replaces it with real transparency, re-encoding the image as
// PNG. The checkerboard is the background region that matches the two
// dominant tones and connects to the image border; motif areas the
// tones touch only inside closed outlines are kept. It reports false,
// leaving the input untouched, when the image already has
// transparency, has no dominant bright neutral tone pair, or the
// border-connected share is implausible for a background.
func stripFakeTransparency(mime string, data []byte) (string, []byte, bool) {
	if mime != "image/png" && mime != "image/jpeg" {
		return mime, data, false
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return mime, data, false
	}
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width < 32 || height < 32 {
		return mime, data, false
	}
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), src, bounds.Min, draw.Src)

	// real transparency anywhere means there is nothing to repair
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] < 250 {
			return mime, data, false
		}
	}

	toneA, toneB, ok := checkerTones(img)
	if !ok {
		return mime, data, false
	}
	classes := classifyCheckerPixels(img, toneA, toneB)
	cleared := clearConnectedBackground(img, classes, width, height)
	if share := float64(cleared) / float64(width*height); share < 0.2 || share > 0.97 {
		return mime, data, false
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return mime, data, false
	}
	return "image/png", buf.Bytes(), true
}

// clearConnectedBackground flood-fills from the image border over the
// classified tone pixels, makes the reached pixels transparent and
// returns how many were cleared.
func clearConnectedBackground(img *image.NRGBA, classes []uint8, width, height int) int {
	visited := make([]bool, len(classes))
	queue := make([]int, 0, width*height/4)
	push := func(idx int) {
		if classes[idx] != 0 && !visited[idx] {
			visited[idx] = true
			queue = append(queue, idx)
		}
	}
	for x := 0; x < width; x++ {
		push(x)
		push((height-1)*width + x)
	}
	for y := 0; y < height; y++ {
		push(y * width)
		push(y*width + width - 1)
	}
	cleared := 0
	for len(queue) > 0 {
		idx := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		img.Pix[idx*4+3] = 0
		cleared++
		x, y := idx%width, idx/width
		if x > 0 {
			push(idx - 1)
		}
		if x < width-1 {
			push(idx + 1)
		}
		if y > 0 {
			push(idx - width)
		}
		if y < height-1 {
			push(idx + width)
		}
	}
	return cleared
}

// checkerTones finds the two dominant tones of the image and reports
// whether they look like a checkerboard pair: both bright, both
// neutral, distinct in luminance and jointly covering at least
// checkerMinCoverage of the pixels.
func checkerTones(img *image.NRGBA) (toneA, toneB [3]int, ok bool) {
	type bucket struct {
		count int
		sumR  int
		sumG  int
		sumB  int
	}
	buckets := make(map[uint32]*bucket)
	for i := 0; i < len(img.Pix); i += 4 {
		r, g, b := int(img.Pix[i]), int(img.Pix[i+1]), int(img.Pix[i+2])
		key := uint32(r>>3)<<10 | uint32(g>>3)<<5 | uint32(b>>3)
		entry, found := buckets[key]
		if !found {
			entry = &bucket{}
			buckets[key] = entry
		}
		entry.count++
		entry.sumR += r
		entry.sumG += g
		entry.sumB += b
	}
	var top [2]*bucket
	for _, entry := range buckets {
		switch {
		case top[0] == nil || entry.count > top[0].count:
			top[1] = top[0]
			top[0] = entry
		case top[1] == nil || entry.count > top[1].count:
			top[1] = entry
		}
	}
	if top[1] == nil {
		return toneA, toneB, false
	}
	centroid := func(entry *bucket) [3]int {
		return [3]int{entry.sumR / entry.count, entry.sumG / entry.count, entry.sumB / entry.count}
	}
	toneA, toneB = centroid(top[0]), centroid(top[1])
	total := len(img.Pix) / 4
	if float64(top[0].count+top[1].count) < checkerMinCoverage*float64(total) {
		return toneA, toneB, false
	}
	for _, tone := range [2][3]int{toneA, toneB} {
		lo := min(tone[0], tone[1], tone[2])
		hi := max(tone[0], tone[1], tone[2])
		if lo < 190 || hi-lo > 12 {
			return toneA, toneB, false
		}
	}
	lumA := (toneA[0] + toneA[1] + toneA[2]) / 3
	lumB := (toneB[0] + toneB[1] + toneB[2]) / 3
	if diff := abs(lumA - lumB); diff < 10 || diff > 90 {
		return toneA, toneB, false
	}
	return toneA, toneB, true
}

// classifyCheckerPixels labels every pixel 1 for tone A, 2 for tone B
// and 0 for anything else. A pixel joins a tone when it is neutral
// and within checkerToneTolerance of the centroid; borderline pixels
// go to the nearer centroid.
func classifyCheckerPixels(img *image.NRGBA, toneA, toneB [3]int) []uint8 {
	classes := make([]uint8, len(img.Pix)/4)
	for i := 0; i < len(img.Pix); i += 4 {
		r, g, b := int(img.Pix[i]), int(img.Pix[i+1]), int(img.Pix[i+2])
		if max(r, g, b)-min(r, g, b) > checkerMaxSpread {
			continue
		}
		distA := toneDistance(r, g, b, toneA)
		distB := toneDistance(r, g, b, toneB)
		switch {
		case distA <= checkerToneTolerance && distA <= distB:
			classes[i/4] = 1
		case distB <= checkerToneTolerance:
			classes[i/4] = 2
		}
	}
	return classes
}

func toneDistance(r, g, b int, tone [3]int) int {
	return max(abs(r-tone[0]), abs(g-tone[1]), abs(b-tone[2]))
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
