package utils

import (
	"crypto/md5"
	"encoding/hex"
	"math"
	"strconv"
	"strings"
)

func HasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}
func HasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// FloatPrecisionStr float 转换为 string 精度转换
func FloatPrecisionStr(f float64, prec int, round bool) string {
	ff := Precision(f, prec, round)
	str := strconv.FormatFloat(ff, 'f', prec, 64)

	return str
}

// Precision 支持精度以及是否四舍五入, round: true 为四舍五入, false 不是四舍五入
func Precision(f float64, prec int, round bool) float64 {
	// 需要加上对长度的校验, 否则直接用 math.Trunc 会有bug(1.14会变成1.13)
	arr := strings.Split(strconv.FormatFloat(f, 'f', -1, 64), ".")
	if len(arr) < 2 {
		return f
	}
	if len(arr[1]) <= prec {
		return f
	}
	pow10N := math.Pow10(prec)

	if round {
		return math.Trunc((f+0.5/pow10N)*pow10N) / pow10N
	}

	return math.Trunc((f)*pow10N) / pow10N
}

// 字符串是否包含在字符数组里
func IsStringInArray(target string, strArray []string) bool {
	for _, element := range strArray {
		if target == element {
			return true
		}
	}
	return false
}

// 数值是否包含在数字组里
func IsIntInArray(target int, intArray []int) bool {
	for _, element := range intArray {
		if target == element {
			return true
		}
	}
	return false
}

// 字符串MD5加密
func Md5EncodeToString(s string) string {
	hexCode := md5.Sum([]byte(s))
	return hex.EncodeToString(hexCode[:])
}

func Overlay(str string, overlay string, start int, end int) string {
	if str == "" {
		return ""
	} else {
		strLen := len(str)
		if start < 0 {
			start = 0
		}
		if start > strLen {
			start = strLen
		}
		if end < 0 {
			end = 0
		}
		if end > strLen {
			end = strLen
		}
		if start > end {
			temp := start
			start = end
			end = temp
		}
		return Substring(str, 0, start) + overlay + Substring(str, end, strLen)
	}
}

func Substring(source string, start int, end int) string {
	var r = []rune(source)
	length := len(r)
	if start < 0 || end > length || start > end {
		return ""
	}
	if start == 0 && end == length {
		return source
	}
	return string(r[start:end])
}

// 字符串在另外一个字符串里，出现第num次的位置
func OrdinalIndexOf(source, str string, num int) int {
	var r = []rune(source)
	lenStr := len(r)
	variable := -1
	if num <= 0 {
		return variable
	}
	for i := 0; i < lenStr; i++ {
		if string(r[i]) == str {
			variable++
		}
		if variable == num {
			return i
		}
	}
	return variable
}
