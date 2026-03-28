package utils

import (
	"path/filepath"
	"regexp"
	"strings"
)

//url域名正侧
const domainRegex string = "^(http(s)?):(.)+(com|cn|net|org|biz|info|cc|tv)"

//处理富文本 匹配img正则
const tagRegex = "<\\s*img\\s+([^>]*)\\s*"
const tagAttrib4Regex = "src=\\S*\"([^\"]+)\""
const pureUrlRegex = "\"([^\"]*)\"" //获取双引号之间的内容

//拼接域名 相对路径
func BindUrl(hostname string, path ...string) string {
	if hostname == "" {
		return strings.TrimLeft(filepath.Join(path...), "/")
	}
	if len(path) == 0 {
		return hostname
	}
	pathx := &strings.Builder{}
	for _, v := range path {
		newV := strings.TrimRight(v, "/")
		if !strings.HasPrefix(newV, "/") {
			pathx.WriteString("/")
		}
		pathx.WriteString(newV)
	}
	pr := strings.TrimRight(hostname, "/")
	return pr + pathx.String()
}

//替换资源域名
func ReplaceHost(src string, replaceHost string) string {
	if src == "" {
		return src
	}
	if !IsValidHost(src) {
		return src
	}
	reg, err := regexp.Compile(domainRegex)
	if err != nil {
		return src
	}
	return reg.ReplaceAllString(src, strings.TrimRight(replaceHost, "/"))
}

//是否是合法的域名
func IsValidHost(hostPath string) bool {
	reg, err := regexp.Compile(domainRegex)
	if err != nil {
		return false
	}
	return reg.MatchString(hostPath)
}

/**
 *处理指定的资源路径
 * filed-待处理的路径
 * domain-给定的域名 带http开头的
 * filtersArr-不需要处理的域名白名单
 * 如果filed是绝对路径,且其域名 不在白名单内 则替换域名则使用新域名替换,否则不替换
 * 如果是相对路径,则拼接域名domain
 */

func BindOrReplacePath(filed, domain string, filtersArr []string) string {
	if filed == "" {
		return filed
	}

	spiltStr := ","
	urls := strings.Split(filed, spiltStr)
	var newFiled []string
	for _, v := range urls {
		if v == "" {
			continue
		}
		//如果是绝对路径,则使用新域名替换
		if IsValidHost(v) {
			//是否为白名单 不在白名单内 则替换域名
			if !ContainsAnyIgnoreCase(v, filtersArr...) {
				v = ReplaceHost(v, domain)
			}
		} else {
			//相对路径 则拼接域名
			v = BindUrl(domain, v)
		}
		newFiled = append(newFiled, v)
	}
	return strings.Join(newFiled, spiltStr)
}

/**
 * 将富富文本内容，判断是否位白名单，通过整体替换正确路径
 * filed-待处理的路径
 * domain-给定的域名 带http开头的
 * filtersArr-不需要处理的域名白名单
 * 如果filed是绝对路径,且其域名 不在白名单内 则替换域名则使用新域名替换,否则不替换
 * 如果是相对路径,则拼接域名domain
 */
func ReplaceHtmlTags(html, domain string, filtersArr []string) string {
	if html == "" || domain == "" {
		return html
	}
	//查找匹配<img src=""
	reg, err := regexp.Compile(tagRegex)
	if err != nil {
		return html
	}
	//要替换的的匹配正则
	replacReg, err := regexp.Compile(tagAttrib4Regex)
	if err != nil {
		return html
	}

	//获取双引号之间的内容
	pureUrlReg, err := regexp.Compile(pureUrlRegex)
	if err != nil {
		return html
	}

	flag := reg.MatchString(html)
	if !flag {
		return html
	}
	urlOld := reg.FindAllString(html, -1)
	for _, v := range urlOld {
		replaceFind := replacReg.MatchString(v)
		if replaceFind {
			oldAttributeStr := replacReg.FindString(v)
			oldPureAttributeStr := pureUrlReg.FindString(oldAttributeStr)
			oldPureAttributeStr = strings.Trim(oldPureAttributeStr, "\"")
			if oldPureAttributeStr != "" {
				var newAttributeStr string
				//处理  “/” 开头的域名
				if strings.HasPrefix(oldPureAttributeStr, "/") {
					newAttributeStr = BindUrl(domain, oldPureAttributeStr)
					//处理  “http” 开头的域名
				} else if strings.HasPrefix(oldPureAttributeStr, "http") && !ContainsAnyIgnoreCase(oldPureAttributeStr, filtersArr...) {
					//如果域名不是存在白名单种 由于域名后面的图片位第三个“/”的位置所以拿到对应索引截取对应字符串儿
					//替换域名
					newAttributeStr = ReplaceHost(oldPureAttributeStr, domain)
				} else if !strings.HasPrefix(oldPureAttributeStr, "http") && !strings.HasPrefix(oldPureAttributeStr, "/") {
					//处理相对路径非“/”开头的数据
					newAttributeStr = BindUrl(domain, oldPureAttributeStr)
				} else {
					continue
				}
				html = strings.ReplaceAll(html, oldPureAttributeStr, newAttributeStr)
			}
		}
	}
	return html
}
