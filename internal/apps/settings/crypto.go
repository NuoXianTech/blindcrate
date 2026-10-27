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

package settings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"github.com/NuoXianTech/blindcrate/internal/config"
)

// secretAADPrefix 加密时绑定 "设置组.字段" 作为附加数据, 防止密文被挪到其他字段解密
const secretAADPrefix = "blindcrate:settings:"

// secretKeyInfo HKDF 的用途标识, app.secret_key 用于其他用途时派生出的密钥互不相同
const secretKeyInfo = "blindcrate:settings:secret"

// secretCipher 用 HKDF 从 app.secret_key 派生 AES-256-GCM 密钥
func secretCipher() (cipher.AEAD, error) {
	if config.Config.App.SecretKey == "" {
		return nil, errors.New(SecretKeyMissing)
	}
	key, err := hkdf.Key(sha256.New, []byte(config.Config.App.SecretKey), nil, secretKeyInfo, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// encryptSecret 加密密钥字段, 输出 base64(nonce|密文)
//
// field 为 "设置组.字段", 作为附加数据参与认证, 密文挪到其他字段会解密失败。
func encryptSecret(plaintext, field string) (string, error) {
	gcm, err := secretCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), []byte(secretAADPrefix+field))
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// decryptSecret 解密 encryptSecret 的输出
func decryptSecret(ciphertext, field string) (string, error) {
	gcm, err := secretCipher()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], []byte(secretAADPrefix+field))
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
