package requests

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sairson/crawlergo/internal/engine/httplib"
	"github.com/sairson/crawlergo/pkg/urllib"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RequestOptions struct {
	Proxy         string // 请求代理
	Timeout       int    // 请求超时
	Retry         bool   // 重试
	VerifySSL     bool   // 释否验证ssl,默认为false
	AllowRedirect bool   // 是否允许跳转
}

type Session struct {
	RequestOptions
	client *http.Client
}

// generateSessionByOptions 生成请求配配置
func generateSessionByOptions(options *RequestOptions) *Session {
	// 生成默认配置
	if options == nil {
		options = &RequestOptions{
			Timeout:       5,
			VerifySSL:     false,
			AllowRedirect: false,
		}
	}
	timeout := time.Duration(options.Timeout) * time.Second
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !options.VerifySSL},
	}
	if options.Proxy != "" {
		proxyUrl, err := url.Parse(options.Proxy)
		if err == nil {
			tr.Proxy = http.ProxyURL(proxyUrl)
		}
	}
	client := &http.Client{Timeout: timeout, Transport: tr}
	if !options.AllowRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return &Session{RequestOptions: RequestOptions{
		Proxy:         options.Proxy,
		Timeout:       options.Timeout,
		Retry:         options.Retry,
		VerifySSL:     options.VerifySSL,
		AllowRedirect: options.AllowRedirect,
	}, client: client}
}

// Get GET请求
func Get(url string, headers map[string]string, options *RequestOptions) (*httplib.ResponseCrawler, error) {
	return generateSessionByOptions(options).doRequest("GET", url, headers, nil)
}

// Request 自定义请求类型
func Request(verb string, url string, headers map[string]string, body []byte, options *RequestOptions) (*httplib.ResponseCrawler, error) {
	return generateSessionByOptions(options).doRequest(verb, url, headers, body)
}

// Get Session的Get类型请求
func (session *Session) Get(url string, headers map[string]string) (*httplib.ResponseCrawler, error) {
	return session.doRequest("GET", url, headers, nil)
}

// Post Session的Post类型请求
func (session *Session) Post(url string, headers map[string]string, body []byte) (*httplib.ResponseCrawler, error) {
	return session.doRequest("POST", url, headers, body)
}

// Request Session的自定义请求类型
func (session *Session) Request(verb string, url string, headers map[string]string, body []byte) (*httplib.ResponseCrawler, error) {
	return session.doRequest(verb, url, headers, body)
}

func (session *Session) doRequest(verb string, url string, headers map[string]string, body []byte) (*httplib.ResponseCrawler, error) {
	verb = strings.ToUpper(verb)
	bodyReader := bytes.NewReader(body)
	req, err := http.NewRequest(verb, url, bodyReader)
	if err != nil {
		// 多数情况下是url中包含%
		url = urllib.EscapePercentSign(url)
		req, err = http.NewRequest(verb, url, bodyReader)
	}
	if err != nil {
		return nil, errors.Wrap(err, "build request error")
	}

	// 设置headers头
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	// 设置默认的headers头
	defaultHeaders := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko)" +
			" Chrome/76.0.3809.132 Safari/537.36 C845D9D38B3A68F4F74057DB542AD252 tx/2.0",
		"Range":      fmt.Sprintf("bytes=0-%d", 10240), // 默认获取的最大相应内容，100一般适用绝大部分场景
		"Connection": "close",
	}
	for key, value := range defaultHeaders {
		if _, ok := headers[key]; !ok {
			req.Header.Set(key, value)
		}
	}
	// 设置Host头
	if host, ok := headers["Host"]; ok {
		req.Host = host
	}
	// 设置默认的Content-Type头
	if verb == "POST" && headers["Content-Type"] == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		// 应该手动设置Referer、Origin、和X-Requested-With字段
	}
	// 覆盖Connection头
	req.Header.Set("Connection", "close")

	// 请求
	var resp *http.Response
	for i := 0; i <= 0; i++ {
		resp, err = session.client.Do(req)
		if err != nil {
			continue
		} else {
			break
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "error occurred during request")
	}
	// 带Range头后一般webserver响应都是206 PARTIAL CONTENT，修正为200 OK
	if resp.StatusCode == 206 {
		resp.StatusCode = 200
		resp.Status = "200 OK"
	}

	if resp.ContentLength > 0 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		_ = resp.Body.Close()
		return &httplib.ResponseCrawler{
			Response: *resp,
			Body:     b,
		}, nil
	}
	return nil, fmt.Errorf("content-Length <= 0")
}
