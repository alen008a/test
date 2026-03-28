// Package libip ip 信息
package libip

import (
	"fmt"
	"msgPushSite/config"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/internal/context"
	"msgPushSite/lib/httpclient"
	"msgPushSite/mdata"
	"msgPushSite/mdata/rediskey"
	"net"
	"strings"
	"time"

	"github.com/ipipdotnet/ipdb-go"
	jsoniter "github.com/json-iterator/go"
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

var (
	ipdbInfo *ipdb.City
	searcher *xdb.Searcher
)

// 初始化 ip 库
func InitIP() error {
	var err error
	fmt.Println("ipdb文件地址", config.GetServiceAddr().IPConfAddr)
	ipdbInfo, err = ipdb.NewCity(config.GetServiceAddr().IPConfAddr)
	if err != nil {
		return err
	}
	// 缓存整个 xdb 数据 ip2region.xdb (IP库版本升级到 2.11.0)
	// 1、从 dbPath 加载整个 xdb 到内存
	cBuff, err := xdb.LoadContentFromFile(config.GetServiceAddr().IP2RegionAddr)
	if err != nil {
		return err
	}
	// 2、用全局的 cBuff 创建完全基于内存的查询对象。
	searcher, err = xdb.NewWithBuffer(cBuff)
	if err != nil {
		return err
	}
	// 备注：并发使用，用整个 xdb 缓存创建的 searcher 对象可以安全用于并发。
	return nil
}

func GetIPLoc(c *context.Context, ip string) *mdata.IPLoc {
	var data mdata.IPLoc
	data.Country = "中国"

	if ipdbInfo == nil {
		c.Errorf("ipdbInfo is nil")
		return &data
	}
	//IPV6格式IP
	if strings.Contains(ip, ":") {
		if config.GetApplication().SiteAliIpUrlOpen {
			url := config.GetServiceAddr().SiteAliIpUrlV2
			key := config.GetServiceAddr().SiteAliIPKeyV2
			res := FindIpLocationV2(c, url, key, ip)
			if res != nil && res.Province != "" {
				if config.GetApplication().RecordRunLogTimeOpen {
					c.Infof(">>> res=%s -- ipv6=%s", mdata.MustMarshal2String(res), ip)
				}
				return res
			}
		}
		url := config.GetConfig().SiteIpv6Url
		key := config.GetConfig().SiteIpv6Key
		res := findIpv6Location(c, url, key, ip)
		return res
	}

	loc, err := ipdbInfo.FindInfo(ip, "CN")
	if err != nil {
		c.Errorf("ip=%s query err: %v", ip, err)
		return &data
	}

	data.Country = loc.CountryName
	data.Province = loc.RegionName
	data.City = loc.CityName

	if data.Province == "中国" || data.Province == "" || data.Country == data.Province {
		if config.GetApplication().SiteAliIpUrlOpen {
			url := config.GetServiceAddr().SiteAliIpUrlV2
			key := config.GetServiceAddr().SiteAliIPKeyV2
			res := FindIpLocationV2(c, url, key, ip)
			if res != nil && res.Province != "" {
				if config.GetApplication().RecordRunLogTimeOpen {
					c.Infof(">>> res=%s -- ipv4=%s", mdata.MustMarshal2String(res), ip)
				}
				return res
			}
		}
		dataV2 := GetIpLocV2(c, ip)
		if dataV2 != nil && data.Country != data.Province {
			return dataV2
		} else {
			url := config.GetConfig().SiteAliIpv4Url
			key := config.GetConfig().SiteAliIpv4Key
			dataV3 := FindIpv4Location(c, url, key, ip)
			if dataV3 != nil {
				return dataV3
			}
		}
	}

	return &data
}

func GetIpLocV2(c *context.Context, ip string) (data *mdata.IPLoc) {
	ipData, err := searcher.SearchByStr(ip)
	if err != nil {
		c.Errorf(">>> GetIpLocV2 -- err=%v -- ip=%v", err, ip)
		return data
	}

	ipDataArr := strings.Split(ipData, "|")
	if len(ipDataArr) > 0 && ipDataArr[0] != "0" && ipDataArr[1] != "0" {
		data = &mdata.IPLoc{
			Country:  ipDataArr[0],
			Province: ipDataArr[1],
		}
		if ipDataArr[2] != "0" {
			data.City = ipDataArr[2]
		}
	}
	return
}

func findIpv6Location(c *context.Context, url, key, ip string) *mdata.IPLoc {
	timeValue := time.Now()
	res := &mdata.IPLoc{}
	if url == "" || key == "" {
		c.Errorf("ipv6配置错误,url=%s,key=%s,ip=%s", url, key, ip)
		return res
	}
	ipUrl := fmt.Sprintf("%s?key=%s&ip=%s", url, key, ip)
	ipInfo, err := httpclient.POSTJson(ipUrl, []byte(""), nil, httpclient.GetShortProxyClient(2*time.Second))
	if err != nil {
		c.Errorf("IPV6查询失败 err=%v,url=%s", err, ipUrl)
		return res
	}
	c.Errorf("IPV6查询日志 err=%v,url=%s", err, ipUrl)
	var ipJson struct {
		Code string `json:"code"`
		Data struct {
			Country  string `json:"country"`
			Province string `json:"province"`
			City     string `json:"city"`
		}
	}
	err = jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(ipInfo, &ipJson)
	if err != nil || ipJson.Code != "Success" {
		c.Errorf("IPV6查询失败 err=%v,url=%s,resp=%+v", err, ipUrl, string(ipInfo))
		return res
	}
	if config.GetApplication().RecordRunLogTimeOpen {
		c.Infof(">>> findIpv6Location 查询完成时间 -- time:%v -- res=%s", time.Since(timeValue), mdata.MustMarshal2String(res))
	}
	res.City = ipJson.Data.City
	res.Country = ipJson.Data.Country
	res.Province = ipJson.Data.Province
	return res
}

func FindIpv4Location(c *context.Context, url, key, ip string) *mdata.IPLoc {
	// 查询redis是否存在 存在则取redis 不存在则查询 记录redis
	timeValue := time.Now()
	res := &mdata.IPLoc{}
	memKey := fmt.Sprintf(rediskey.LoginParseIpRecords, ip)
	isExists, err := core.KeyExist(false, memKey)
	if err != nil {
		c.Errorf(">>> redis key=%s -- err=%v", memKey, err)
	} else {
		if isExists {
			item, err := core.GetKey(false, memKey)
			if err != nil {
				c.Errorf(">>> redis get key=%s -- err=%v", memKey, err)
			} else {
				err = mdata.Cjson.UnmarshalFromString(item, &res)
				if err != nil {
					c.Errorf(">>> json unmarshalFromString err=%v -- item=%s -- redis key=%s", err, item, memKey)
				} else {
					if config.GetEnv() == "dev" || config.GetEnv() == "pre" {
						c.Infof(">>> redis find success! -- ip=%s -- redis key=%s -- result=%+v", ip, memKey, res)
					}
					return res
				}
			}
		}
	}

	if url == "" || key == "" {
		c.Errorf("ipv4配置错误,url=%s,key=%s,ip=%s", url, key, ip)
		return res
	}
	ipUrl := fmt.Sprintf("%s?ip=%s", url, ip)
	headerMap := make(map[string]string)
	headerMap["Authorization"] = fmt.Sprintf("APPCODE %s", key)
	ipInfo, err := httpclient.ProxyGet(ipUrl, headerMap, httpclient.GetShortProxyClient(time.Second*10))
	if err != nil {
		c.Errorf("IPV4查询失败 err=%v,url=%s", err, ipUrl)
		return res
	}
	var ipJson mdata.AliIpJson
	err = mdata.Cjson.Unmarshal(ipInfo, &ipJson)
	if err != nil || ipJson.Msg != "success" {
		c.Errorf("IPV4查询失败 err=%v,url=%s,resp=%+v", err, ipUrl, string(ipInfo))
		return res
	}
	if config.GetEnv() == "dev" || config.GetEnv() == "pre" {
		c.Infof(">>> ip=%s -- ipJson=%+v", ip, ipJson)
	}
	res.Country = ipJson.Data.Country
	res.Province = ipJson.Data.Region
	res.City = ipJson.Data.City
	saveJson, err := mdata.Cjson.MarshalToString(res)
	if err != nil {
		c.Errorf(">>> ip json save redis fail! ip=%s -- err=%v -- res=%+v", ip, err, res)
	} else {
		// ip地址结果集保留30天
		err = core.SetExpireKV(memKey, saveJson, 30*24*time.Hour)
		if err != nil {
			c.Errorf(">>> redis save fail! -- err=%v -- res=%s -- redis key=%s", err, mdata.MustMarshal2String(res), memKey)
		}
	}
	if config.GetApplication().RecordRunLogTimeOpen {
		c.Infof(">>> findIpv6Location 查询完成时间 -- time:%v -- res=%s", time.Since(timeValue), mdata.MustMarshal2String(res))
	}
	return res
}

func FindIpLocationV2(c *context.Context, url, key, ip string) *mdata.IPLoc {
	// 查询redis是否存在 存在则取redis 不存在则查询 记录redis
	timeValue := time.Now()
	res := &mdata.IPLoc{}
	memKey := fmt.Sprintf(rediskey.LoginParseIpRecords, ip)
	isExists, err := core.KeyExist(false, memKey)
	if err != nil {
		c.Errorf(">>> redis key=%s -- err=%v", memKey, err)
	} else {
		if isExists {
			item, err := core.GetKey(false, memKey)
			if err != nil {
				c.Errorf(">>> redis get key=%s -- err=%v", memKey, err)
			} else {
				err = mdata.Cjson.UnmarshalFromString(item, &res)
				if err != nil {
					c.Errorf(">>> json unmarshalFromString err=%v -- item=%s -- redis key=%s", err, item, memKey)
				} else {
					if config.GetEnv() == "dev" || config.GetEnv() == "pre" {
						c.Infof(">>> redis find success! -- ip=%s -- redis key=%s -- result=%+v", ip, memKey, res)
					}
					return res
				}
			}
		}
	}

	if url == "" || key == "" {
		c.Errorf("ipv4|ipv6配置错误,url=%s,key=%s,ip=%s", url, key, ip)
		return res
	}
	ipUrl := fmt.Sprintf("%s?ip=%s", url, ip)
	headerMap := make(map[string]string)
	headerMap["Authorization"] = fmt.Sprintf("APPCODE %s", key)
	ipInfo, err := httpclient.ProxyGet(ipUrl, headerMap, httpclient.GetShortProxyClient(time.Second*10))
	if err != nil {
		c.Errorf("IPV4|IPV6查询失败 err=%v,url=%s", err, ipUrl)
		return res
	}
	var ipJson mdata.AliIpJsonV2
	err = mdata.Cjson.Unmarshal(ipInfo, &ipJson)
	if err != nil {
		c.Errorf("IPV4|IPV6查询失败 err=%v,url=%s,resp=%+v", err, ipUrl, string(ipInfo))
		return res
	}
	if config.GetEnv() == "dev" || config.GetEnv() == "pre" {
		c.Infof(">>> ip=%s -- ipJson=%+v", ip, ipJson)
	}
	res.Country = ipJson.Data.Country
	res.Province = ipJson.Data.Prov
	res.City = ipJson.Data.City
	saveJson, err := mdata.Cjson.MarshalToString(res)
	if err != nil {
		c.Errorf(">>> ip json save redis fail! ip=%s -- err=%v -- res=%+v", ip, err, res)
	} else {
		// ip地址结果集保留30天
		err = core.SetExpireKV(memKey, saveJson, 30*24*time.Hour)
		if err != nil {
			c.Errorf(">>> redis save fail! -- err=%v -- res=%s -- redis key=%s", err, mdata.MustMarshal2String(res), memKey)
		}
	}
	if config.GetApplication().RecordRunLogTimeOpen {
		c.Infof(">>> FindIpLocationV2 查询完成时间 -- time:%v -- res=%s", time.Since(timeValue), mdata.MustMarshal2String(res))
	}
	return res
}

// ClientIP 更换获取IP方式
func ClientIP(c *context.Context) string {
	clientIP := c.GetHeader("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(c.GetHeader("X-Real-Ip"))
	}
	if clientIP != "" {
		return clientIP
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}
