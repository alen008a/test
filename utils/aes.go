package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

// Aes cbc 加密, pkcs7, base64编码 填充
func AesCBCPk7EncryptBase64(origData, key []byte, iv []byte) (string, error) {
	encryptBytes, err := AesCBCPk7Encrypt(origData, key, iv)
	if err != nil {
		return "", err
	}
	str := base64.StdEncoding.EncodeToString(encryptBytes)

	return str, nil
}

// Aes cbc 解密, pkcs7 填充, base64编码
func AesCBCPk7DecryptBase64(encrypt string, key []byte, iv []byte) (string, error) {
	encryptBytes, err := base64.StdEncoding.DecodeString(encrypt)
	if err != nil {
		return "", err
	}

	str, err := AesCBCPk7Decrypt(encryptBytes, key, iv)
	if err != nil {
		return "", err
	}

	return str, nil
}

// Aes cbc 加密, pkcs7 填充, hex编码
func AesCBCPk7EncryptHex(origData, key []byte, iv []byte) (string, error) {
	encryptBytes, err := AesCBCPk7Encrypt(origData, key, iv)
	if err != nil {
		return "", err
	}
	str := hex.EncodeToString(encryptBytes)

	return str, nil
}

// Aes cbc 解密, pkcs7 填充, hex编码
func AesCBCPk7DecryptHex(encrypt string, key []byte, iv []byte) (string, error) {
	encryptBytes, err := hex.DecodeString(encrypt)
	if err != nil {
		return "", err
	}

	str, err := AesCBCPk7Decrypt(encryptBytes, key, iv)
	if err != nil {
		return "", err
	}

	return str, nil
}

// Aes cbc 加密, pkcs5, base64编码 填充
func AesCBCPk5EncryptBase64(origData, key []byte, iv []byte) (string, error) {
	encryptBytes, err := AesCBCPk5Encrypt(origData, key, iv)
	if err != nil {
		return "", err
	}
	str := base64.StdEncoding.EncodeToString(encryptBytes)

	return str, nil
}

// Aes cbc 解密, pkcs5 填充, base64编码
func AesCBCPk5DecryptBase64(encrypt string, key []byte, iv []byte) (string, error) {
	encryptBytes, err := base64.StdEncoding.DecodeString(encrypt)
	if err != nil {
		return "", err
	}

	str, err := AesCBCPk5Decrypt(encryptBytes, key, iv)
	if err != nil {
		return "", err
	}

	return str, nil
}

// Aes cbc 加密, pkcs5 填充, hex编码
func AesCBCPk5EncryptHex(origData, key []byte, iv []byte) (string, error) {
	encryptBytes, err := AesCBCPk5Encrypt(origData, key, iv)
	if err != nil {
		return "", err
	}
	str := hex.EncodeToString(encryptBytes)

	return str, nil
}

// Aes cbc 解密, pkcs7 填充, hex编码
func AesCBCPk5DecryptHex(encrypt string, key []byte, iv []byte) (string, error) {
	encryptBytes, err := hex.DecodeString(encrypt)
	if err != nil {
		return "", err
	}

	str, err := AesCBCPk5Decrypt(encryptBytes, key, iv)
	if err != nil {
		return "", err
	}

	return str, nil
}

// Aes cbc 加密, pkcs7 填充
func AesCBCPk7Encrypt(orgData, key []byte, iv []byte) ([]byte, error) {
	if len(orgData) < 1 {
		return []byte(""), errors.New("orgData is empty")
	}
	if len(key) < 1 {
		return []byte(""), errors.New("key is empty")
	}
	if len(iv) < 1 {
		return []byte(""), errors.New("iv is empty")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	orgData = PKCS7Padding(orgData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, iv[:blockSize])
	encryption := make([]byte, len(orgData))
	blockMode.CryptBlocks(encryption, orgData)
	return encryption, nil
}

// Aes cbc 解密, pkcs7 填充
func AesCBCPk7Decrypt(encryption, key []byte, iv []byte) (string, error) {
	if len(encryption) < 1 {
		return "", errors.New("encryption is empty")
	}
	if len(key) < 1 {
		return "", errors.New("key is empty")
	}
	if len(iv) < 1 {
		return "", errors.New("iv is empty")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 加入判断条件防止 panic
	blockSize := block.BlockSize()
	if len(key) < blockSize {
		return "", errors.New("key too short")
	}
	if len(encryption)%blockSize != 0 {
		return "", errors.New("crypto/cipher: input not full blocks")
	}

	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(encryption))
	blockMode.CryptBlocks(origData, encryption)
	origData = PKCS7UnPadding(origData, blockSize)
	return string(origData), nil
}

// Aes cbc 加密, pkcs5 填充
func AesCBCPk5Encrypt(orgData, key []byte, iv []byte) ([]byte, error) {
	if len(orgData) < 1 {
		return []byte(""), errors.New("orgData is empty")
	}
	if len(key) < 1 {
		return []byte(""), errors.New("key is empty")
	}
	if len(iv) < 1 {
		return []byte(""), errors.New("iv is empty")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	orgData = PKCS5Padding(orgData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, iv[:blockSize])
	encryption := make([]byte, len(orgData))
	blockMode.CryptBlocks(encryption, orgData)
	return encryption, nil
}

// Aes cbc 解密, pkcs5 填充
func AesCBCPk5Decrypt(encryption, key []byte, iv []byte) (string, error) {
	if len(encryption) < 1 {
		return "", errors.New("encryption is empty")
	}
	if len(key) < 1 {
		return "", errors.New("key is empty")
	}
	if len(iv) < 1 {
		return "", errors.New("iv is empty")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 加入判断条件防止 panic
	blockSize := block.BlockSize()
	if len(key) < blockSize {
		return "", errors.New("key too short")
	}
	if len(encryption)%blockSize != 0 {
		return "", errors.New("crypto/cipher: input not full blocks")
	}

	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(encryption))
	blockMode.CryptBlocks(origData, encryption)
	origData = PKCS5UnPadding(origData)
	return string(origData), nil
}

// aes加密 ecb模式 模拟Java SHA1PRNG
func AesEncryptECBSHA1PRNG(origData []byte, key []byte) (encrypted []byte, err error) {
	key, _ = AesSha1prng(key, 128)
	cipher, err := aes.NewCipher(generateKey(key))
	if err != nil {
		return nil, err
	}
	length := (len(origData) + aes.BlockSize) / aes.BlockSize
	plain := make([]byte, length*aes.BlockSize)
	copy(plain, origData)
	pad := byte(len(plain) - len(origData))
	for i := len(origData); i < len(plain); i++ {
		plain[i] = pad
	}
	encrypted = make([]byte, len(plain))
	// 分组分块加密
	for bs, be := 0, cipher.BlockSize(); bs <= len(origData); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Encrypt(encrypted[bs:be], plain[bs:be])
	}

	return encrypted, nil
}

func AesDecryptECBSHA1PRNG(encrypted []byte, key []byte) (decrypted []byte) {
	key, _ = AesSha1prng(key, 128)
	cipher, _ := aes.NewCipher(generateKey(key))
	decrypted = make([]byte, len(encrypted))
	//
	for bs, be := 0, cipher.BlockSize(); bs < len(encrypted); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Decrypt(decrypted[bs:be], encrypted[bs:be])
	}

	trim := 0
	if len(decrypted) > 0 {
		trim = len(decrypted) - int(decrypted[len(decrypted)-1])
	}

	return decrypted[:trim]
}

func generateKey(key []byte) (genKey []byte) {
	genKey = make([]byte, 16)
	copy(genKey, key)
	for i := 16; i < len(key); {
		for j := 0; j < 16 && i < len(key); j, i = j+1, i+1 {
			genKey[j] ^= key[i]
		}
	}
	return genKey
}

func ZeroPadding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{0}, padding)
	return append(ciphertext, padtext...)
}

func ZeroUnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

// PKCS7 填充
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// PKCS7 填充（修复BUG）
func PKCS7UnPadding(plantText []byte, blockSize int) []byte {
	length := len(plantText)
	unpadding := int(plantText[length-1])
	resultLen := length - unpadding
	if resultLen < 0 {
		return []byte("")
	}
	return plantText[:(length - unpadding)]
}

func getAesKey(key string) []byte {
	if len(key) != 32 {
		panic("error secret key")
	}
	return []byte(key[2:7] + key[11:15] + key[18:25])
}

func getIv(key string) []byte {
	if len(key) != 32 {
		panic("error secret key")
	}
	return []byte(key[4:9] + key[16:23] + key[25:29])
}
func ParseKey(secretKey string, secretIv string, secretData string) (str string, err error) {
	data, err := base64.StdEncoding.DecodeString(secretData)
	if err != nil {
		return
	}
	decrypted, err := AesCBCPk7Decrypt(data, []byte(secretKey), []byte(secretIv))
	if err != nil {
		return
	}
	return string(decrypted), nil
}

// SHA1PRNG 处理
func AesSha1prng(keyBytes []byte, encryptLength int) ([]byte, error) {
	hashs := Sha1(Sha1(keyBytes))
	maxLen := len(hashs)
	realLen := encryptLength / 8
	if realLen > maxLen {
		return nil, errors.New("invalid length!")
	}

	return hashs[0:realLen], nil
}

func Sha1(data []byte) []byte {
	h := sha1.New()
	h.Write(data)
	return h.Sum(nil)
}

// AesDecrypt AES解密
func AesDecrypt(crypted, key []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte("")
	}
	blockMode := NewECBDecrypter(block)
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS5UnPadding(origData)
	return origData
}

// AesEncrypt AES加密
func AesEncrypt(src, key string) []byte {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return []byte("")
	}
	if src == "" {
		return []byte("")
	}
	ecb := NewECBEncrypter(block)
	content := []byte(src)
	content = PKCS5Padding(content, block.BlockSize())
	crypted := make([]byte, len(content))
	ecb.CryptBlocks(crypted, content)
	return crypted
}

// PKCS5Padding 填充
func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// PKCS5UnPadding ...
func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	// 去掉最后一个字节 unpadding 次
	unpadding := int(origData[length-1])
	if length <= unpadding {
		return nil
	}
	return origData[:(length - unpadding)]
}

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb

// NewECBEncrypter returns a BlockMode which encrypts in electronic code book
// mode, using the given Block.
func NewECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}
func (x *ecbEncrypter) BlockSize() int { return x.blockSize }
func (x *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Encrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

type ecbDecrypter ecb

// NewECBDecrypter returns a BlockMode which decrypts in electronic code book
// mode, using the given Block.
func NewECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbDecrypter)(newECB(b))
}
func (x *ecbDecrypter) BlockSize() int { return x.blockSize }
func (x *ecbDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Decrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

func GetRealString(encodingAesKey string, data string) string {
	dataTmp, err := hex.DecodeString(data)
	if err != nil {
		panic(fmt.Sprintf("%s 十六进制解码错误，%+v", data, err))
	}

	var md5Str = Md5EncodeToString(encodingAesKey)
	rs, err := AesCBCPk7Decrypt(dataTmp, getAesKey(md5Str), getIv(md5Str))
	if err != nil {
		panic(fmt.Sprintf("%s AES解密错误，%+v", data, err))
	}

	return rs
}

func Base64Encode(src []byte) []byte {
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(dst, src)
	dst = bytes.TrimSpace(dst)
	return dst
}

func Base64Decode(src []byte) (out []byte, err error) {
	dBuf := make([]byte, base64.StdEncoding.DecodedLen(len(src)))
	n, err := base64.StdEncoding.Decode(dBuf, src)
	return dBuf[:n], err
}
