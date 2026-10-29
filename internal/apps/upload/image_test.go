/*
 * MIT License
 *
 * Copyright (c) 2026 nuoxi4n, NuoXianTech
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package upload

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// encode 生成一张 w×h 的测试图片, alpha 为 true 时左上角透明
func encode(t *testing.T, w, h int, format string, alpha bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	if alpha {
		img.Set(0, 0, color.NRGBA{})
	}
	var buf bytes.Buffer
	var err error
	if format == "png" {
		err = png.Encode(&buf, img)
	} else {
		err = jpeg.Encode(&buf, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withOrientation 在 jpeg 的 SOI 之后插入只有方向标签的 EXIF 段
func withOrientation(data []byte, orientation uint16) []byte {
	tiff := []byte("MM\x00\x2a\x00\x00\x00\x08\x00\x01")
	entry := make([]byte, 12)
	binary.BigEndian.PutUint16(entry[0:], 0x0112)
	binary.BigEndian.PutUint16(entry[2:], 3)
	binary.BigEndian.PutUint32(entry[4:], 1)
	binary.BigEndian.PutUint16(entry[8:], orientation)
	payload := append(append([]byte("Exif\x00\x00"), tiff...), entry...)
	payload = append(payload, 0, 0, 0, 0)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	out := append([]byte{}, data[:2]...)
	out = append(append(out, seg...), payload...)
	return append(out, data[2:]...)
}

func TestProcess(t *testing.T) {
	cases := []struct {
		name          string
		data          []byte
		spec          purposeSpec
		contentType   string
		width, height int
	}{
		{"png kept as png", encode(t, 40, 20, "png", false), purposeSpec{maxSide: 512}, contentTypePNG, 40, 20},
		{"jpeg resized", encode(t, 200, 100, "jpeg", false), purposeSpec{maxSide: 64}, contentTypeJPEG, 64, 32},
		{"tall resized", encode(t, 50, 200, "png", true), purposeSpec{maxSide: 100}, contentTypePNG, 25, 100},
		{"jpeg rotated", withOrientation(encode(t, 60, 30, "jpeg", false), 6), purposeSpec{maxSide: 512}, contentTypeJPEG, 30, 60},
		{"jpeg rotated and resized", withOrientation(encode(t, 200, 100, "jpeg", false), 8), purposeSpec{maxSide: 50}, contentTypeJPEG, 25, 50},
		{"square cropped", encode(t, 90, 30, "png", false), purposeSpec{square: true, maxSide: 512}, contentTypePNG, 30, 30},
		{"square cropped and resized", withOrientation(encode(t, 400, 300, "jpeg", false), 6), purposeSpec{square: true, maxSide: 100}, contentTypeJPEG, 100, 100},
	}
	for _, c := range cases {
		got, err := process(c.data, c.spec)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.contentType != c.contentType || got.width != c.width || got.height != c.height {
			t.Errorf("%s: got %s %dx%d, want %s %dx%d", c.name, got.contentType, got.width, got.height, c.contentType, c.width, c.height)
		}
		decoded, _, err := image.DecodeConfig(bytes.NewReader(got.data))
		if err != nil || decoded.Width != c.width || decoded.Height != c.height {
			t.Errorf("%s: saved image is %dx%d, err %v", c.name, decoded.Width, decoded.Height, err)
		}
	}
}

func TestProcessInvalid(t *testing.T) {
	var invalid *InvalidError
	if _, err := process([]byte("not an image"), purposeSpec{maxSide: 512}); !errors.As(err, &invalid) {
		t.Errorf("garbage: got %v", err)
	}
	truncated := encode(t, 100, 100, "png", false)
	if _, err := process(truncated[:len(truncated)/2], purposeSpec{maxSide: 512}); !errors.As(err, &invalid) {
		t.Errorf("truncated: got %v", err)
	}
}

func TestOrientation(t *testing.T) {
	plain := encode(t, 8, 8, "jpeg", false)
	if got := jpegOrientation(plain); got != 1 {
		t.Errorf("no exif: got %d", got)
	}
	for o := uint16(1); o <= 8; o++ {
		if got := jpegOrientation(withOrientation(plain, o)); got != int(o) {
			t.Errorf("orientation %d: got %d", o, got)
		}
	}
	if got := jpegOrientation(withOrientation(plain, 9)); got != 1 {
		t.Errorf("invalid orientation: got %d", got)
	}

	// 2×1 的图片顺时针旋转 90 度后左边的像素在上面
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.NRGBA{R: 255, A: 255})
	src.Set(1, 0, color.NRGBA{B: 255, A: 255})
	dst := applyOrientation(src, 6)
	if b := dst.Bounds(); b.Dx() != 1 || b.Dy() != 2 {
		t.Fatalf("rotated size %v", b)
	}
	if r, _, _, _ := dst.At(0, 0).RGBA(); r == 0 {
		t.Errorf("rotated top pixel is not red")
	}
}

func TestCropSquare(t *testing.T) {
	// 3×1 的图片裁出中间的像素
	src := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	src.Set(1, 0, color.NRGBA{G: 255, A: 255})
	dst := cropSquare(src)
	if b := dst.Bounds(); b.Dx() != 1 || b.Dy() != 1 {
		t.Fatalf("cropped size %v", b)
	}
	if _, g, _, _ := dst.At(dst.Bounds().Min.X, 0).RGBA(); g == 0 {
		t.Errorf("cropped pixel is not the middle one")
	}
}
