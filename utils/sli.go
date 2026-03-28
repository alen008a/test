package utils

import (
	"strings"
)

type BaseType interface {
	int | int32 | int64 | uint | uint8 | uint32 | uint64 | string
}

func ContainsBaseType[T BaseType](slc []T, item T) bool {
	flag := false
	for i := range slc {
		if slc[i] == item {
			flag = true
			break
		}
	}
	return flag
}

// 给任意字符串类型数组插入元素
func InsertStringSlice(slice []string, index int, value ...string) []string {
	var newStr []string
	if index == 0 {
		return append(append(newStr, value...), slice...)
	}
	if index == len(slice)-1 {
		return append(append(newStr, slice...), value...)
	}
	return append(append(append(newStr, slice[0:index]...), value...), slice[index:]...)
}

// 是否int64数组包含
func ContainsInt64(source []int64, des int64) bool {
	for _, val := range source {
		if val == des {
			return true
		}
	}
	return false
}

// 元素去重
func RemoveRep(slc []string) []string {
	if len(slc) < 1024 {
		// 切片长度小于1024的时候，循环来过滤
		return RemoveRepByLoop(slc)
	} else {
		// 大于的时候，通过map来过滤
		return RemoveRepByMap(slc)
	}
}

// 通过map主键唯一的特性过滤重复元素
func RemoveRepByMap(slc []string) []string {
	var result []string
	tempMap := map[string]byte{} // 存放不重复主键
	for _, e := range slc {
		l := len(tempMap)
		tempMap[e] = 0
		if len(tempMap) != l { // 加入map后，map长度变化，则元素不重复
			result = append(result, e)
		}
	}
	return result
}

// 通过两重循环过滤重复元素
func RemoveRepByLoop(slc []string) []string {
	var result []string // 存放结果
	for i := range slc {
		flag := true
		for j := range result {
			if slc[i] == result[j] {
				flag = false // 存在重复元素，标识为false
				break
			}
		}
		if flag { // 标识为false，不添加进结果
			result = append(result, slc[i])
		}
	}
	return result
}

// 是否包含某一个元素
func Contains(slc []string, item string) bool {
	flag := false
	for i := range slc {
		if slc[i] == item {
			flag = true
			break
		}
	}
	return flag
}

func ContainsUint(slc []uint64, item uint64) bool {
	flag := false
	for i := range slc {
		if slc[i] == item {
			flag = true
			break
		}
	}
	return flag
}

// 是否包含某一个内容 不区分大小写
func ContainsAnyIgnoreCase(url string, urls ...string) bool {
	for _, v := range urls {
		lowerUrl := strings.ToLower(url)
		if strings.Contains(lowerUrl, strings.ToLower(v)) {
			return true
		}
	}
	return false
}

func InIntArray(val int, array []int) (exists bool) {
	exists = false
	for _, v := range array {
		if val == v {
			exists = true
			return
		}
	}
	return
}
func InStrArray(val string, array []string) (exists bool) {
	exists = false
	for _, v := range array {
		if val == v {
			exists = true
			return
		}
	}
	return
}
