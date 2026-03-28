package httpclient

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"msgPushSite/config"
	"msgPushSite/mdata"
	"net/http"
	netUrl "net/url"
	"strings"
	"time"
)

var (
	defaultHttpClient *http.Client
	Header            = map[string]string{
		"Content-type": "application/json",
	}
)

// Basic Authentication
type BasicAuth struct {
	Username string
	Password string
}

func POST(path string, data []byte, header map[string]string, basicAuth ...BasicAuth) ([]byte, error) {
	isJson := mdata.Cjson.Valid(data)
	if !isJson {
		return []byte(""), errors.New("请求字符串非json格式！")
	}

	payload := strings.NewReader(string(data))
	req, _ := http.NewRequest("POST", path, payload)
	req.Header.Add("content-type", "application/json")

	if len(basicAuth) > 0 {
		if basicAuth[0].Username != "" && basicAuth[0].Password != "" {
			req.SetBasicAuth(basicAuth[0].Username, basicAuth[0].Password)
		}
	}

	for key, value := range header {
		req.Header.Add(key, value)
	}

	rsp, err := HttpClient.Do(req)
	if err != nil {
		return []byte(""), err
	}
	defer rsp.Body.Close()

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return []byte(""), err
	}

	if rsp.StatusCode == 200 {
		return body, nil
	}

	return []byte(""), nil
}

func GET(path string, header map[string]string, basicAuth ...BasicAuth) ([]byte, error) {

	req, _ := http.NewRequest("GET", path, nil)

	for key, value := range header {
		req.Header.Add(key, value)
	}

	if len(basicAuth) > 0 {
		if basicAuth[0].Username != "" && basicAuth[0].Password != "" {
			req.SetBasicAuth(basicAuth[0].Username, basicAuth[0].Password)
		}
	}

	rsp, err := HttpClient.Do(req)
	if err != nil {
		return []byte(""), err
	}
	defer rsp.Body.Close()

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return []byte(""), err
	}

	return body, nil
}
func POSTJson(path string, data []byte, header map[string]string, cli *http.Client) ([]byte, error) {
	if cli == nil {
		cli = HttpClient
	}
	cli.Timeout = 5 * time.Second
	payload := bytes.NewReader(data)
	req, err := http.NewRequest("POST", path, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Add("content-type", "application/json")

	for key, value := range header {
		req.Header.Add(key, value)
	}

	rsp, err := cli.Do(req)
	if err != nil {
		return []byte(""), err
	}
	defer rsp.Body.Close()

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return []byte(""), err
	}

	return body, nil
}

func ProxyPostJson(path, body string, header map[string]string) ([]byte, error) {
	var s []byte
	proxy := getProxy()
	myClient := &http.Client{Transport: &http.Transport{Proxy: proxy}}

	request, _ := http.NewRequest("POST", path, strings.NewReader(body))
	if len(header) > 0 {
		Header = header
	}
	for key, value := range Header {
		request.Header.Add(key, value)
	}
	res, err := myClient.Do(request)
	if err != nil {
		return nil, err
	}
	if res.Status == "404 " {
		return nil, err
	}
	defer res.Body.Close()
	body1, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return s, err
	}
	return body1, nil
}

// newClient for connection re-use
func getProxy() func(*http.Request) (*netUrl.URL, error) {
	proxy := http.ProxyFromEnvironment
	if len(config.GetApplication().VerifyProxyUrl) != 0 {
		par, err := netUrl.Parse(config.GetApplication().VerifyProxyUrl)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		proxy = http.ProxyURL(par)
	}
	return proxy
}

func ProxyGet(path string, header map[string]string, cli *http.Client) ([]byte, error) {
	if cli == nil {
		cli = HttpClient
	}
	req, _ := http.NewRequest("GET", path, nil)
	for key, value := range header {
		req.Header.Add(key, value)
	}
	rsp, err := cli.Do(req)
	if err != nil {
		return []byte(""), err
	}
	defer rsp.Body.Close()

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return []byte(""), err
	}

	return body, nil
}

func CheckESIndexesExists(path string, header map[string]string, basicAuth ...BasicAuth) (int, error) {
	if path == "" {
		return 400, errors.New("请求地址不能为空!")
	}
	req, _ := http.NewRequest("HEAD", path, nil)
	for key, value := range header {
		req.Header.Add(key, value)
	}

	if len(basicAuth) > 0 {
		if basicAuth[0].Username != "" && basicAuth[0].Password != "" {
			req.SetBasicAuth(basicAuth[0].Username, basicAuth[0].Password)
		}
	}

	var client = HttpClient
	rsp, err := client.Do(req)
	if err != nil {
		return 400, err
	}
	defer rsp.Body.Close()
	_, err = io.ReadAll(rsp.Body)
	if err != nil {
		return 400, err
	}
	return rsp.StatusCode, nil
}
