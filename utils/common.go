package utils

import (
	"math"
	"strings"
)

func NumConvert(memberId int) int {
	return memberId & 63
}

func MaskRealName(realName string) string {
	if realName == "" {
		return ""
	} else {
		strings.Trim(realName, " ")
		return Overlay(realName, "**", 1, len(realName))
	}
}

// MaskIp
func MaskIp(ip string) string {
	if ip == "" {
		return ""
	} else {
		ipArray := strings.Split(ip, ".")
		if len(ipArray) >= 2 {
			return ipArray[0] + "." + ipArray[1] + ".*.*"
		} else {
			return ip
		}
	}
}
func MaskPhone(phone string) string {
	if phone == "" {
		return ""
	} else {
		strings.Trim(phone, " ")
		return Overlay(phone, "****", 3, 7)
	}
}

func MaskEmail(email string) string {
	if email == "" {
		return ""
	} else {
		strings.Trim(email, " ")
		at := "@"
		if !strings.Contains(email, at) {
			return email
		} else {
			options := strings.Split(email, at)
			if len(options) < 2 {
				return email
			} else {
				return Overlay(options[0], "****", 2, len(options[0])) + at + options[1]
			}
		}
	}
}

func MaskQq(qq string) string {
	if qq == "" {
		return ""
	} else {
		strings.Trim(qq, " ")
		return Overlay(qq, "****", 2, len(qq))
	}
}

func MaskAddress(address string) string {
	if address == "" {
		return ""
	} else {
		addressArray := strings.Split(address, ",")
		sb := strings.Builder{}
		if len(addressArray) > 1 {
			sb.WriteString(addressArray[0])
			sb.WriteString(" ")
			sb.WriteString(addressArray[1])
			sb.WriteString(" ****")
			sb.WriteString(" ****")
		} else {
			sb.WriteString(addressArray[0])
			sb.WriteString(" ****")
			sb.WriteString(" ****")
		}
		return sb.String()
	}
}

func MaskBankNum(bankNum string) string {
	if bankNum == "" {
		return ""
	} else {
		return Overlay(bankNum, "**** **** **** ", 0, len(bankNum)-4)
	}
}

func PageNUms(total, pageSize int) (pageNums int) {
	if total < 1 || pageSize < 1 {
		return
	}
	pageNums = int(math.Ceil(float64(total) / float64(pageSize)))
	return
}

// 数组用
func PageOffsetAndEnd(total, pageSize, page int) (pageNums, offset, end int) {
	pageNums = PageNUms(total, pageSize)
	if pageNums < 1 || page > pageNums {
		return
	}
	offset = (page - 1) * pageSize
	end = offset + pageSize
	if end > total {
		end = total
	}
	return
}

func IsFinger(data string) bool {
	return len(data) == 36 && HasSuffix(data, "FPFP")
}

// 判断字符串至少有一个不为空
func IsLeastOne(s ...string) bool {
	for _, v := range s {
		if v != "" {
			return true
		}
	}
	return false
}
