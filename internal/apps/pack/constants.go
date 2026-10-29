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

package pack

// 配置包的格式
const (
	packFormat   = "blindcrate-pack" // manifest.json 中的 format
	packVersion  = 1                 // 当前的格式版本, 格式变化时加 1, 并在 packUpgrades 中补上旧版本的升级函数
	packManifest = "manifest.json"
	packImageDir = "images/" // 打包的上传图片放在这个目录下, 路径为 images/<key>, 配置中的图片地址也写成这个路径
)

// 配置包的限制
const (
	maxPackSize      = 64 << 20 // 配置包文件的大小上限
	maxPackJSONSize  = 16 << 20 // 包中单个 JSON 文件解压后的大小上限
	maxPackImageSize = 20 << 20 // 包中单张图片的大小上限, 与上传图片的大小上限一致
	maxPackFiles     = 5000     // 包中的文件数上限
	maxPackEntries   = 2000     // 每种配置的条数上限
	maxPackImages    = 4900     // 导出时的图片数上限, 给 manifest 和配置文件留出位置, 不超过 maxPackFiles
	packFormOverhead = 1 << 20  // 上传表单中除文件以外的部分
)

// 配置的种类, 也是包中的文件名(加上 .json)
const (
	PackCurrencies   = "currencies"
	PackQualities    = "qualities"
	PackItems        = "items"
	PackLootPools    = "loot_pools"
	PackVenues       = "venues"
	PackShopItems    = "shop_items"
	PackLeaderboards = "leaderboards"
	PackSettings     = "settings" // 只有游戏分类的设置组, 不含密钥字段
)

// packSections 全部配置种类, 按依赖排列, 导入时按这个顺序写入
var packSections = []string{PackCurrencies, PackQualities, PackItems, PackLootPools, PackVenues, PackShopItems, PackLeaderboards, PackSettings}

// packSectionLabels 配置种类的名称, 用于提示
var packSectionLabels = map[string]string{
	PackCurrencies:   "货币",
	PackQualities:    "品质",
	PackItems:        "物品",
	PackLootPools:    "掉落池",
	PackVenues:       "地图",
	PackShopItems:    "商品",
	PackLeaderboards: "榜单",
	PackSettings:     "设置",
}

// 导入时一项配置与已有配置的比较结果
const (
	PackStatusNew      = "new"      // 没有对应的配置, 导入时新增
	PackStatusChanged  = "changed"  // 有对应的配置且内容不同, 导入时覆盖
	PackStatusSame     = "same"     // 有对应的配置且内容相同, 不需要导入
	PackStatusConflict = "conflict" // 无法导入, 原因见 reason, 需要跳过或改名导入
)

// 导入时一项配置的处理方式
const (
	PackActionOverwrite = "overwrite" // 新增, 或覆盖对应的配置; 没有指定处理方式时的默认值
	PackActionSkip      = "skip"      // 跳过, 不导入
	PackActionRename    = "rename"    // 以新的代码导入, 包中引用它的配置一起改为新代码; 只有掉落池、地图、商品和榜单可以改名
)
