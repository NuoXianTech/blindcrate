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
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // 注册 webp 解码
)

// processed 重新编码后的图片
type processed struct {
	data        []byte
	contentType string
	ext         string
	width       int
	height      int
}

func (p *processed) reader() io.Reader { return bytes.NewReader(p.data) }

// process 解码原图, 按用途裁剪和缩小, 按 EXIF 方向转正后重新编码
//
// 裁剪和缩小都以中心对称, 先在原方向上处理再转正, 转正时处理的像素更少;
// 原图有透明通道或为 png 时保存为 png(保留像素画等图片的清晰边缘), 其他保存为 jpeg;
// 重新编码只保留像素, EXIF(含拍摄地点)等元数据和图片之后附带的数据都会去掉。
func process(data []byte, spec purposeSpec) (*processed, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, &InvalidError{Reason: ImageInvalid}
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, &InvalidError{Reason: ImageInvalid}
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return nil, &InvalidError{Reason: fmt.Sprintf(ImageTooLarge, maxPixels/10_000)}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, &InvalidError{Reason: ImageInvalid}
	}
	alpha := !isOpaque(img)
	if spec.square {
		img = cropSquare(img)
	}
	img = resize(img, spec.maxSide)
	if format == "jpeg" {
		img = applyOrientation(img, jpegOrientation(data))
	}

	out := &processed{width: img.Bounds().Dx(), height: img.Bounds().Dy()}
	var buf bytes.Buffer
	if alpha || format == "png" {
		err = png.Encode(&buf, img)
		out.contentType, out.ext = contentTypePNG, ".png"
	} else {
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality})
		out.contentType, out.ext = contentTypeJPEG, ".jpg"
	}
	if err != nil {
		return nil, err
	}
	out.data = buf.Bytes()
	return out, nil
}

// isOpaque 图片是否完全不透明, 无法判断时按有透明处理
func isOpaque(img image.Image) bool {
	o, ok := img.(interface{ Opaque() bool })
	return ok && o.Opaque()
}

// cropSquare 从中间裁出最大的正方形
func cropSquare(src image.Image) image.Image {
	b := src.Bounds()
	sub, ok := src.(interface {
		SubImage(r image.Rectangle) image.Image
	})
	if b.Dx() == b.Dy() || !ok {
		return src
	}
	side := min(b.Dx(), b.Dy())
	x := b.Min.X + (b.Dx()-side)/2
	y := b.Min.Y + (b.Dy()-side)/2
	return sub.SubImage(image.Rect(x, y, x+side, y+side))
}

// resize 等比缩小到最大边长 maxSide 以内, 不放大
func resize(src image.Image, maxSide int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSide && h <= maxSide {
		return src
	}
	if w >= h {
		w, h = maxSide, max(1, h*maxSide/w)
	} else {
		w, h = max(1, w*maxSide/h), maxSide
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	return dst
}

// applyOrientation 按 EXIF 方向(1-8)旋转或翻转图片, 使其正向显示; 重新编码后不再带方向, 需要先转正
func applyOrientation(src image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if orientation >= 5 {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch orientation {
			case 2: // 水平翻转
				dx, dy = w-1-x, y
			case 3: // 旋转 180 度
				dx, dy = w-1-x, h-1-y
			case 4: // 垂直翻转
				dx, dy = x, h-1-y
			case 5: // 沿主对角线翻转
				dx, dy = y, x
			case 6: // 顺时针旋转 90 度
				dx, dy = h-1-y, x
			case 7: // 沿副对角线翻转
				dx, dy = h-1-y, w-1-x
			case 8: // 逆时针旋转 90 度
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

// jpegOrientation 读取 jpeg 的 EXIF 方向, 没有或无法识别时返回 1
//
// 依次读取图像数据之前的段, 找到 APP1 中的 EXIF, 读取第一个 IFD 中的方向标签 0x0112。
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		switch {
		case marker == 0xFF: // 填充字节
			i++
			continue
		case marker == 0xDA || marker == 0xD9: // 图像数据开始或结束
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2:]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		if marker == 0xE1 {
			if o := exifOrientation(data[i+4 : i+2+size]); o != 0 {
				return o
			}
		}
		i += 2 + size
	}
	return 1
}

// exifOrientation 从 APP1 段的内容中读取方向, 不是 EXIF 或没有方向时返回 0
func exifOrientation(seg []byte) int {
	if len(seg) < 14 || string(seg[:6]) != "Exif\x00\x00" {
		return 0
	}
	tiff := seg[6:]
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}
	if order.Uint16(tiff[2:]) != 42 {
		return 0
	}
	ifd := int(order.Uint32(tiff[4:]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 0
	}
	n := int(order.Uint16(tiff[ifd:]))
	for k := 0; k < n; k++ {
		entry := ifd + 2 + k*12
		if entry+12 > len(tiff) {
			return 0
		}
		if order.Uint16(tiff[entry:]) != 0x0112 {
			continue
		}
		// 类型需为 SHORT(3), 值在值字段的前 2 个字节
		if order.Uint16(tiff[entry+2:]) != 3 {
			return 0
		}
		if o := int(order.Uint16(tiff[entry+8:])); o >= 1 && o <= 8 {
			return o
		}
		return 0
	}
	return 0
}
