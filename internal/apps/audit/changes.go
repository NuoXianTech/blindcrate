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

package audit

import (
	"bytes"
	"encoding/json"
	"reflect"
)

// Changes 比较修改前后的两组字段, 返回内容变了的字段, 值为 {"old": 旧值, "new": 新值}, 用作修改类操作的 detail
//
// old 和 next 由调用方按同一份字段清单生成, 只比较 old 中有的字段。按 JSON 的内容比较:
// 切片、map 和结构体按内容, 原始 JSON(json.RawMessage)不受键的顺序和空白影响, 从数据库读出的 JSONB 与请求中的写法不同时不算修改。
// 时间需由调用方统一成同一时区的字符串, 否则同一时刻的不同时区算作修改。没有变化时返回空 map。
func Changes(old, next map[string]interface{}) map[string]interface{} {
	changes := map[string]interface{}{}
	for key, oldValue := range old {
		if newValue := next[key]; !sameContent(oldValue, newValue) {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}

// sameContent 两个值序列化成 JSON 后内容是否相同; 无法序列化时按值比较
func sameContent(a, b interface{}) bool {
	x, errA := canonical(a)
	y, errB := canonical(b)
	if errA != nil || errB != nil {
		return reflect.DeepEqual(a, b)
	}
	return reflect.DeepEqual(x, y)
}

// canonical 把值序列化成 JSON 再解析, 得到与写法无关的内容; 数字保留原文, 不经过 float64, 大整数不丢精度
func canonical(v interface{}) (interface{}, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out interface{}
	if err := decoder.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
