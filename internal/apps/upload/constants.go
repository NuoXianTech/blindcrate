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

// Package upload 图片上传: 头像、物品图片、地图封面和商品图片
//
//   - 上传: 按内容识别格式, 只接受 png、jpeg、webp, 按用途缩小尺寸后重新编码(去掉 EXIF 等元数据), 见 Save
//   - 存储: 文件按内容的 sha256 命名, 同一用途的相同图片只存一份, 业务表只保存返回的地址, 见 storage
//   - 清理: 上传后超过保留天数仍没有被引用的图片定时删除, 引用的位置见 references, 见 HandleCleanup
package upload

// 图片的用途, 决定上传权限和缩小后的最大边长
const (
	PurposeAvatar = "avatar" // 头像, 登录用户都可以上传
	PurposeItem   = "item"   // 物品图片, 只有管理员可以上传
	PurposeVenue  = "venue"  // 地图封面, 只有管理员可以上传
	PurposeShop   = "shop"   // 商品图片, 只有管理员可以上传
)

// purposeSpec 用途的上传限制
type purposeSpec struct {
	adminOnly bool // 只有管理员可以上传
	square    bool // 从中间裁成正方形, 头像按圆形或方形展示, 不拉伸
	maxSide   int  // 缩小后的最大边长(像素), 保持宽高比
}

var purposes = map[string]purposeSpec{
	PurposeAvatar: {square: true, maxSide: 256},
	PurposeItem:   {adminOnly: true, maxSide: 512},
	PurposeVenue:  {adminOnly: true, maxSide: 1280},
	PurposeShop:   {adminOnly: true, maxSide: 512},
}

// 保存的图片格式, 原图有透明通道或为 png 时保存为 png, 其他保存为 jpeg
const (
	contentTypePNG  = "image/png"
	contentTypeJPEG = "image/jpeg"
	contentTypeWebP = "image/webp"
)

const (
	// maxPixels 原图最多的像素数, 解码前按图片头检查, 避免解码超大图片耗尽内存
	maxPixels = 40_000_000
	// jpegQuality 保存为 jpeg 时的质量
	jpegQuality = 90
	// multipartOverhead 请求体除图片外允许的大小, 用于 multipart 的分隔符和其他字段
	multipartOverhead = 64 << 10
	// cleanupBatch 清理时每批检查的上传数
	cleanupBatch = 100
)
