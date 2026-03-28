package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Flag string

const (
	Alpha  Flag = "alpha"  //判断字符串是不是英文字母
	Digit  Flag = "digit"  //判断字符串是不是数字
	AlNum  Flag = "alNum"  //判断字符串是不是字母+数字
	String Flag = "string" //判断字符串
	Bool   Flag = "bool"   //判断字符串 是不是布尔型
	Mail   Flag = "mail"   //校验邮箱
	Float  Flag = "float"  //判断字符串 是不是浮点型
	Empty  Flag = "empty"  //判断字符串是不是为空串
)

type Validator struct {
	Min   int
	Max   int
	Field string
	Value string
	Flags Flag
}

func CheckEmail(str string) bool {
	ma, err := regexp.MatchString("^[A-Za-z\\d]+([-_.][A-Za-z\\d]+)*@([A-Za-z\\d]+[-.])+[A-Za-z\\d]{2,4}$", str)
	if err != nil {
		return false
	}
	return ma
}

func checkBool(str string) bool {
	_, err := strconv.ParseBool(str)
	if err != nil {
		return false
	}
	return true
}

func checkFloat(str string) bool {
	_, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return false
	}
	return true
}

func checkLength(str string, min, max int) bool {
	if min == 0 && max == 0 {
		return true
	}

	n := len(str)
	if n < min || n > max {
		return false
	}
	return true
}

func Verify(rules []Validator) (string, bool) {
	for _, val := range rules {
		if val.Flags == Alpha && (!IsAlpha(val.Value) || !checkLength(val.Value, val.Min, val.Max)) {
			return val.Field, false
		} else if val.Flags == Digit && (!IsDigit(val.Value) || !checkLength(val.Value, val.Min, val.Max)) {
			return val.Field, false
		} else if val.Flags == AlNum && (!IsAlNum(val.Value) || !checkLength(val.Value, val.Min, val.Max)) {
			return val.Field, false
		} else if val.Flags == String && !checkLength(val.Value, val.Min, val.Max) {
			return val.Field, false
		} else if val.Flags == Bool && !checkBool(val.Value) {
			return val.Field, false
		} else if val.Flags == Mail && !CheckEmail(val.Value) {
			return val.Field, false
		} else if val.Flags == Float && !checkFloat(val.Value) {
			return val.Field, false
		} else if val.Flags == Empty && IsEmpty(val.Value) {
			return val.Field, false
		}

	}

	return "", true
}

//IsEmpty 判断字符串是不是为空串
func IsEmpty(s string) bool {
	return s == ""
}

// IsAlpha 判断字符串是不是英文字母
func IsAlpha(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isAlpha(r) {
			return false
		}
	}
	return true
}

// IsDigit 判断字符串是不是数字
func IsDigit(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isDigit(r) {
			return false
		}
	}
	return true
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

// IsAlNum 判断字符串是不是字母+数字
func IsAlNum(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isDigit(r) && !isAlpha(r) {
			return false
		}
	}
	return true
}

func isAlpha(r rune) bool {
	if r >= 'A' && r <= 'Z' {
		return true
	} else if r >= 'a' && r <= 'z' {
		return true
	}
	return false
}

// CheckFunc 校验函数
// minLen: 最小长度
// maxLen: 最大长度
// str: 需要校验的字符串
// regexpStr: 正则表达式
// return error: 错误
func CheckFunc(minLen, maxLen int, str, regexpStr string) error {
	strLen := len(str)
	if strLen < minLen || strLen > maxLen {
		return fmt.Errorf("the length is invalid, %v-%v", minLen, maxLen)
	}

	// 判断正则表达式是否有误
	regCom, err := regexp.Compile(regexpStr)
	if err != nil {
		tmpStr := fmt.Sprintf("expression of regexp=%v is err: %v", regexpStr, err)
		return errors.New(tmpStr)
	}

	// 对 string 进行校验
	matchFlag := regCom.MatchString(str)
	if !matchFlag {
		tmpStr := fmt.Sprintf("params not match, is invalid")
		return errors.New(tmpStr)
	}

	return nil
}

// NumLetter 数字和字母
func NumLetter(minLen, maxLen int, str string) error {
	regexpStr := "^[a-zA-Z0-9]*$"

	return CheckFunc(minLen, maxLen, str, regexpStr)
}

// NumCheck 数字
func NumCheck(minLen, maxLen int, str string) error {
	regexpStr := "^[0-9]*$"

	return CheckFunc(minLen, maxLen, str, regexpStr)
}

// NetterCheck 英文字母
func NetterCheck(minLen, maxLen int, str string) error {
	regexpStr := "^[a-zA-Z]*$"

	return CheckFunc(minLen, maxLen, str, regexpStr)
}

// 中国大陆手机号码验证
func ChinaPhoneCheck(minLen, maxLen int, phone string) error {
	regexpStr := "^1[0-9]*$"
	return CheckFunc(minLen, maxLen, phone, regexpStr)
}

// 验证qq
func CheckQQ(minLen, maxLen int, str string) error {
	regexpStr := "^[1-9][0-9]*$"

	return CheckFunc(minLen, maxLen, str, regexpStr)
}

// PwdCheck 密码校验, 以数字和字母开头,包含下划线和扛
func PwdCheck(minLen, maxLen int, str string) error {
	regexpStr := "^[a-zA-Z0-9][a-zA-Z0-9_-]*$"

	return CheckFunc(minLen, maxLen, str, regexpStr)
}

// UUIDCheck uuid 校验, 以数字和字母开头,包含下划线和扛
func UUIDCheck(minLen, maxLen int, str string) error {
	regexpStr := "^[a-zA-Z0-9][a-zA-Z0-9_-]*$"

	return CheckFunc(minLen, maxLen, str, regexpStr)
}

func CheckRealName(realName string, min int, max int) bool {
	realName = strings.Trim(realName, "")
	count := utf8.RuneCountInString(realName)
	if count < min || count > max {
		return false
	}
	matchCn, errCn := regexp.MatchString("^[\u4e00-\u9fa5]+([·•][\u4e00-\u9fa5]+)*$", realName)
	matchEn, errEn := regexp.MatchString("^[a-zA-Z]+([\\s·•]?[a-zA-Z]+)+$", realName)
	if (!matchCn || errCn != nil) && (!matchEn || errEn != nil) {
		return false
	}
	return true
}
func CheckIp(ip string) bool {
	ma, err := regexp.MatchString("([1-9]|[1-9]\\d|1\\d{2}|2[0-4]\\d|25[0-5])(\\.(\\d|[1-9]\\d|1\\d{2}|2[0-4]\\d|25[0-5])){3}", ip)
	if err != nil {
		return false
	}
	return ma
}
