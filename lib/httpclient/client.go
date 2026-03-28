package httpclient

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"msgPushSite/config"
)

var (
	HttpClient *http.Client
)

// init HTTPClient  默认开启长链接(http 1.1之后) 开启http keepalive功能，也即是否重用连接，
func init() {
	HttpClient = newClient(true)
}

const (
	MaxIdleConnNum     int = 1500 // 连接池对所有host的最大链接数量，host也即dest-ip, 默认 100
	MaxIdleConnPerHost int = 1500 // 连接池对每个host的最大链接数量
	IdleConnTimeout    int = 90   // 空闲timeout设置，也即socket在该时间内没有交互则自动关闭连接（注意：该timeout起点是从每次空闲开始计时，若有交互则重置为0）,该参数通常设置为分钟级别
	Timeout            int = 30   // 请求以及连接的超时时间
)

func newClient(verifyFlag bool) *http.Client {
	if verifyFlag != true {
		return &http.Client{
			Transport: &http.Transport{
				Proxy:           http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				DialContext: (&net.Dialer{
					Timeout:   time.Duration(Timeout) * time.Second,
					KeepAlive: time.Duration(Timeout) * time.Second,
				}).DialContext,

				MaxIdleConns:        MaxIdleConnNum,
				MaxIdleConnsPerHost: MaxIdleConnPerHost,
				IdleConnTimeout:     time.Duration(IdleConnTimeout) * time.Second,
			},

			Timeout: time.Duration(Timeout) * time.Second,
		}
	} else {
		return &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   time.Duration(Timeout) * time.Second,
					KeepAlive: time.Duration(Timeout) * time.Second,
				}).DialContext,

				MaxIdleConns:        MaxIdleConnNum,
				MaxIdleConnsPerHost: MaxIdleConnPerHost,
				IdleConnTimeout:     time.Duration(IdleConnTimeout) * time.Second,
			},

			Timeout: time.Duration(Timeout) * time.Second,
		}
	}
}

// GetClient 获取 http client
func GetClient(verifyFlag bool, timeout time.Duration) *http.Client {
	return createNotMultiplexHTTPClient(verifyFlag, timeout)
}

// newClient for connection re-use
func createNotMultiplexHTTPClient(verifyFlag bool, timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = time.Duration(Timeout) * time.Second
	}

	if !verifyFlag {
		return &http.Client{
			Transport: &http.Transport{
				Proxy:           http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				DialContext: (&net.Dialer{
					Timeout:   timeout,
					KeepAlive: time.Duration(Timeout) * time.Second,
				}).DialContext,
				IdleConnTimeout: time.Duration(IdleConnTimeout) * time.Second,
			},

			Timeout: timeout,
		}
	} else {
		return &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   timeout,
					KeepAlive: time.Duration(Timeout) * time.Second,
				}).DialContext,

				IdleConnTimeout: time.Duration(IdleConnTimeout) * time.Second,
			},

			Timeout: timeout,
		}
	}
}

// GetShortProxyClient 获取 短连接http client
func GetShortProxyClient(timeout time.Duration) *http.Client {
	p, err := url.Parse(config.GetApplication().ProxyURL)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyURL(p),
			DisableKeepAlives: true,
		},
		Timeout: timeout,
	}
}

func GetShortProxyNotifyClient(timeout time.Duration) *http.Client {
	proxyUrl := config.GetApplication().VerifyProxyUrl
	if proxyUrl == "" {
		proxyUrl = config.GetApplication().ProxyURL
	}
	p, err := url.Parse(proxyUrl)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyURL(p),
			DisableKeepAlives: true,
		},
		Timeout: timeout,
	}
}
