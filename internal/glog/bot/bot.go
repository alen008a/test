package bot

import (
	"fmt"
	"msgPushSite/config"
	redisdb "msgPushSite/db/redisdb/core"
	"msgPushSite/lib/httpclient"
	"msgPushSite/mdata"
	"strconv"
	"sync"
	"time"
)

// 告警的前提是不影响业务
var store sync.Map                      //只存代码位置，减少内存占用
var warningMsg = make(chan string, 100) //增加容量，防止业务慢
var cacheKey = "SITE_BOT_WARNING_CACHE_%s"
var timeout int64 = 600

func SendDefault(template, position, msg string) {
	Send(template, "初始化站点", position, msg)
}

func Send(siteId, template, position, msg string) {

	if config.GetApplication().WarningCode == "" {
		return
	}

	var (
		now   = time.Now().Unix()
		t, ok = store.Load(position)
	)
	key := fmt.Sprintf(cacheKey, siteId) + position
	//相同的代码位置，十分钟只推送一次，防止大批量推送导致系统奔溃和漏掉通知
	//如果不存在就从redis去获取,如果redis存在，然后刷到内存，并直接返回
	if !ok {
		rs, err := redisdb.GetKey(false, key)
		if rs != "" && err == nil {
			t1, _ := strconv.ParseInt(rs, 10, 64)
			if t1 > 0 && now-t1 < timeout {
				//刷到内存里
				store.Store(position, t1)
				return
			}
		}
	}

	if ok && now-t.(int64) < timeout {
		return
	}

	//刷新时间
	if ok && now-t.(int64) > timeout {
		store.Delete(position)
	}

	select {
	case <-time.After(time.Millisecond * 100): //不能因为量大的时候阻塞主要业务
	case warningMsg <- fmt.Sprintf(template, config.GetApplication().AppID, siteId+config.GetApplication().Cluster, position, msg):
	}

	//如果是第一次就存入
	if !ok {
		store.Store(position, now)
		redisdb.SetExpireKV(key, strconv.FormatInt(now, 10), time.Second*time.Duration(timeout))
	}
}

func SendMid(siteId, msg, path string) {

	if config.GetApplication().WarningCode == "" {
		return
	}

	//相同的代码位置，十分钟只推送一次，防止大批量推送导致系统奔溃和漏掉通知
	var (
		now   = time.Now().Unix()
		t, ok = store.Load(path)
	)
	key := fmt.Sprintf(cacheKey, siteId) + path
	//如果不存在就从redis去获取,如果redis存在，然后刷到内存，并直接返回
	if !ok {
		rs, err := redisdb.GetKey(false, key)
		if rs != "" && err == nil {
			t1, _ := strconv.ParseInt(rs, 10, 64)
			if t1 > 0 && now-t1 < timeout {
				//刷到内存里
				store.Store(path, t1)
				return
			}
		}
	}

	if ok && now-t.(int64) < timeout {
		return
	}

	//刷新时间
	if ok && now-t.(int64) > timeout {
		store.Delete(path)
	}

	select {
	case <-time.After(time.Millisecond * 100): //不能因为量大的时候阻塞主要业务
	case warningMsg <- msg:
	}

	//如果是第一次就存入
	if !ok {
		store.Store(path, now)
		redisdb.SetExpireKV(key, strconv.FormatInt(now, 10), time.Second*time.Duration(timeout))
	}
}

func send2bot(serviceCode, msg string, botType int) (string, error) {
	data, err := httpclient.POSTJson(
		config.GetApplication().VerifyCodeDomain+"/verifycode/bot/v1/send",
		mdata.MustMarshal(map[string]interface{}{
			"serviceCode": serviceCode,
			"botType":     botType,
			"msg":         msg,
		}),
		map[string]string{"Content-Type": "application/json"},
		nil,
	)
	return string(data), err
}

func init() {
	go watchDog()
}

func watchDog() {
	go func() {
		for v := range warningMsg {
			data, err := send2bot(config.GetApplication().WarningCode, v, 3)
			if err != nil {
				_ = fmt.Errorf("send2bot occur err=%v |data=%v |msg=%s", err, data, v)
			}
		}
	}()
}

const SlackTemplate = `
=============系统警告，开发人员需要注意=============
项目：【%s】
站点：【%s】
代码位置：
	%s
告警内容：
	%s

`

const SlowTemplate = `
-------------慢接口警告，开发人员需要注意-------------
项目：【%s】
站点：【%s】
接口地址：%s
接口耗时：%.2fs
日志追踪：%s


`
