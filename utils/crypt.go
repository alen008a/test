/*
*

	@note:

*
*/
package utils

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

/**
 * sha256加密
 */
func Sha256Encode(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	return fmt.Sprintf("%x", h.Sum(nil))
}

/**
 * hmac-sha256 加密
 */
func HmacSha256Encode(value string, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(value))
	return fmt.Sprintf("%x", h.Sum(nil))
}

/**
 * md5 加密
 */
func Md5Encry(value string) string {
	h := md5.New()
	h.Write([]byte(value))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func MD5EncryByByte(value []byte) string {
	h := md5.New()
	h.Write(value)
	res := h.Sum(nil)
	return hex.EncodeToString(res)
}
