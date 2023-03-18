package engine

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	enums2 "github.com/sairson/crawlergo/internal/engine/enums"
	"github.com/sairson/crawlergo/internal/engine/httplib"
	"github.com/sairson/crawlergo/internal/engine/httplib/urllib"
	"github.com/sairson/crawlergo/internal/engine/requests"
	"github.com/sairson/crawlergo/internal/option"
	"github.com/sairson/crawlergo/pkg/utils"
	"io"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 主要是tab页请求处理

// GetCDPExecutor 获取浏览器tab页上下文
func (tab *Tab) GetCDPExecutor() context.Context {
	c := chromedp.FromContext(*tab.Context)
	ctx := cdp.WithExecutor(*tab.Context, c.Target)
	return ctx
}

// AddTabRequestToResultList 将我们拦截到的tab页请求添加到结果列表当中
func (tab *Tab) AddTabRequestToResultList(req *httplib.RequestCrawler) {
	for key, value := range tab.ExtraHeaders {
		req.Headers[key] = value
	}
	tab.Lock.Lock()
	tab.ResultList = append(tab.ResultList, req)
	if tab.ResultCallback != nil {
		_ = tab.ResultCallback(req) // 执行结果回调
	}
	tab.Lock.Unlock()
}

// AddResultFormCustomUrl 添加一个成果从自定义url
func (tab *Tab) AddResultFormCustomUrl(method string, _url string, source string) {
	navUrl := tab.NavigateRequest.URL
	url, err := urllib.GetURL(_url, *navUrl)
	if err != nil {
		return
	}
	crawlerOption := httplib.OptionsCrawler{
		Headers:  map[string]interface{}{},
		PostData: "",
	}
	referer := navUrl.String()
	// 处理Host绑定
	if host, ok := tab.NavigateRequest.Headers["Host"]; ok {
		if host != navUrl.Hostname() && url.Hostname() == host {
			url, _ = urllib.GetURL(strings.Replace(url.String(), "://"+url.Hostname(), "://"+navUrl.Hostname(), -1), *navUrl)
			crawlerOption.Headers["Host"] = host
			referer = strings.Replace(navUrl.String(), navUrl.Host, host.(string), -1)
		}
	}
	// 添加Cookie
	if cookie, ok := tab.NavigateRequest.Headers["Cookie"]; ok {
		crawlerOption.Headers["Cookie"] = cookie
	}
	// 修正Referer
	crawlerOption.Headers["Referer"] = referer
	for key, value := range tab.ExtraHeaders {
		crawlerOption.Headers[key] = value
	}
	req := httplib.GetCrawlerRequest(method, url, crawlerOption)
	req.Source = source
	tab.Lock.Lock()
	// 直接将结果添加到结果列表
	tab.ResultList = append(tab.ResultList, req)
	if tab.ResultCallback != nil {
		_ = tab.ResultCallback(req)
	}
	tab.Lock.Unlock()
}

// InterceptTabRequest 拦截tab页的请求
func (tab *Tab) InterceptTabRequest(v *fetch.EventRequestPaused) {
	defer tab.WaitGroup.Done() // 完成后释放
	// 先获取tab页的请求上下文
	ctx := tab.GetCDPExecutor()
	// 这里我们拦截url
	url, err := urllib.GetURL(v.Request.URL, *tab.NavigateRequest.URL)
	if err != nil {
		// 如果处理url失败,我们继续请求不做任何处理
		_ = fetch.ContinueRequest(v.RequestID).Do(ctx)
		return
	}
	// 我们记录我们拦截到的请求头和post data数据
	_option := httplib.OptionsCrawler{
		Headers:  v.Request.Headers,
		PostData: v.Request.PostData,
	}
	// 通过这个option,我们生成一个爬虫请求
	crawlerRequest := httplib.GetCrawlerRequest(v.Request.Method, url, _option)
	// 判断请求是否包含需要忽略的字符
	if IsIgnoredByKeywordMatch(*crawlerRequest, tab.config.IgnoreKeywords) {
		// 包含需要忽略的关键字
		_ = fetch.FailRequest(v.RequestID, network.ErrorReasonBlockedByClient).Do(ctx) // 造成请求失败
		crawlerRequest.Source = enums2.FromXHR                                         // ajax异步请求
		tab.AddTabRequestToResultList(crawlerRequest)
		return
	}
	// 这一步我们要不要做host绑定？
	tab.HandleHostBinding(crawlerRequest)

	// 静态资源处理,我们是否要阻断静态资源
	if option.StaticSuffixSet.Contains(url.FileExt()) {
		// 如果我们忽略静态资源,我们就阻断这个请求并将请求添加到我们的结果列表当中
		_ = fetch.FailRequest(v.RequestID, network.ErrorReasonBlockedByClient).Do(ctx)
		crawlerRequest.Source = enums2.FromStaticRes
		tab.AddTabRequestToResultList(crawlerRequest)
		return
	}
	// 处理导航请求
	if tab.IsNavigatorRequest(v.NetworkID.String()) {
		tab.NavNetworkID = v.NetworkID.String()
		tab.HandlerCrawlerNavigationRequest(crawlerRequest, v) // 处理导航请求
		// 添加结果
		crawlerRequest.Source = enums2.FromNavigation
		tab.AddTabRequestToResultList(crawlerRequest)
		return
	}
	crawlerRequest.Source = enums2.FromXHR
	tab.AddTabRequestToResultList(crawlerRequest)
	_ = fetch.ContinueRequest(v.RequestID).Do(ctx)
}

// HandleHostBinding 将请求与我们的Navigate request做绑定
func (tab *Tab) HandleHostBinding(req *httplib.RequestCrawler) {
	url := req.URL
	navUrl := tab.NavigateRequest.URL
	// 导航请求的域名和HOST绑定中的域名不同，且当前请求的domain和导航请求header中的Host相同，则替换当前请求的domain并绑定Host
	if host, ok := tab.NavigateRequest.Headers["Host"]; ok {
		if navUrl.Hostname() != host && url.Host == host {
			urlObj, _ := urllib.GetURL(strings.Replace(req.URL.String(), "://"+url.Hostname(), "://"+navUrl.Hostname(), -1), *navUrl)
			req.URL = urlObj
			req.Headers["Host"] = host

		} else if navUrl.Hostname() != host && url.Host == navUrl.Host {
			req.Headers["Host"] = host
		}
		// 修正Origin
		if _, ok := req.Headers["Origin"]; ok {
			req.Headers["Origin"] = strings.Replace(req.Headers["Origin"].(string), navUrl.Host, host.(string), 1)
		}
		// 修正Referer
		if _, ok := req.Headers["Referer"]; ok {
			req.Headers["Referer"] = strings.Replace(req.Headers["Referer"].(string), navUrl.Host, host.(string), 1)
		} else {
			req.Headers["Referer"] = strings.Replace(navUrl.String(), navUrl.Host, host.(string), 1)
		}
	}
}

// IsNavigatorRequest 判断是否是导航请求
func (tab *Tab) IsNavigatorRequest(networkID string) bool {
	return networkID == tab.LoaderID
}

// HandlerCrawlerNavigationRequest 控制爬虫导航请求
func (tab *Tab) HandlerCrawlerNavigationRequest(req *httplib.RequestCrawler, v *fetch.EventRequestPaused) {

	navReq := tab.NavigateRequest // 我们当前控制的请求
	ctx := tab.GetCDPExecutor()
	tCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	overrideRequest := fetch.ContinueRequest(v.RequestID).WithURL(req.URL.String())
	if tab.FoundRedirection && tab.IsTopFrame(v.FrameID.String()) { // 处理标签页的重定向标记
		body := base64.StdEncoding.EncodeToString([]byte(`<html><body>crawlergo</body></html>`))
		param := fetch.FulfillRequest(v.RequestID, 200).WithBody(body)
		_ = param.Do(ctx)
		// 不对错误做处理
		navReq.Redirection = true
		navReq.Source = enums2.FromNavigation
		// 将重定向请求添加到结果列表
		tab.AddTabRequestToResultList(&navReq)
	} else if navReq.Redirection && tab.IsTopFrame(v.FrameID.String()) {
		// 处理请求的重定向标记
		navReq.Redirection = false
		headers := utils.ConvertHeaders(req.Headers)
		headers["Range"] = "bytes=0-1048576"
		resp, err := requests.Request(req.Method, req.URL.String(), headers, []byte(req.PostData), &requests.RequestOptions{
			AllowRedirect: false, Proxy: tab.config.Proxy,
		})
		if err != nil {
			_ = fetch.FailRequest(v.RequestID, network.ErrorReasonConnectionAborted).Do(ctx)
			return
		}
		body := base64.StdEncoding.EncodeToString([]byte(resp.ToText()))
		// 获取请求参数
		param := fetch.FulfillRequest(v.RequestID, 200).WithResponseHeaders(tab.HeadersNoLocationHeader(resp.Header)).WithBody(body)
		_ = param.Do(ctx)
	} else if tab.IsTopFrame(v.FrameID.String()) && req.URL.NavigationUrl() == navReq.URL.NavigationUrl() {
		// 手动设置POST的信息
		if navReq.Method == enums2.POST || navReq.Method == enums2.PUT {
			overrideRequest = overrideRequest.WithPostData(navReq.PostData)
		}
		overrideRequest = overrideRequest.WithMethod(navReq.Method)
		overrideRequest = overrideRequest.WithHeaders(tab.HeadersMerge(navReq.Headers, req.Headers))
		_ = overrideRequest.Do(tCtx)
	} else if !tab.IsTopFrame(v.FrameID.String()) {
		_ = overrideRequest.Do(tCtx)
	} else {
		// 前端类型跳转,返回204
		_ = fetch.FulfillRequest(v.RequestID, 204).Do(ctx)
	}
}

func (tab *Tab) IsTopFrame(FrameID string) bool {
	return FrameID == tab.TopFrameId
}

// HeadersNoLocationHeader 填充请求头,不带Location标志
func (tab *Tab) HeadersNoLocationHeader(h map[string][]string) []*fetch.HeaderEntry {
	var headers []*fetch.HeaderEntry
	for key, value := range h {
		if key == "Location" {
			continue
		}
		var header fetch.HeaderEntry
		header.Name = key
		header.Value = value[0]
		headers = append(headers, &header)
	}
	return headers
}

// HeadersMerge 合并请求头
func (tab *Tab) HeadersMerge(navHeaders map[string]interface{}, headers map[string]interface{}) []*fetch.HeaderEntry {
	var mergedHeaders []*fetch.HeaderEntry
	for key, value := range navHeaders {
		if _, ok := headers[key]; !ok {
			var header fetch.HeaderEntry
			header.Name = key
			header.Value = value.(string)
			mergedHeaders = append(mergedHeaders, &header)
		}
	}
	for key, value := range headers {
		var header fetch.HeaderEntry
		header.Name = key
		header.Value = value.(string)
		mergedHeaders = append(mergedHeaders, &header)
	}
	return mergedHeaders
}

// ParseRequestURLFormResponseText 解析请求的url从获取的返回体当中
func (tab *Tab) ParseRequestURLFormResponseText(v *network.EventResponseReceived) {
	defer tab.WaitGroup.Done()
	ctx := tab.GetCDPExecutor()
	resp, err := network.GetResponseBody(v.RequestID).Do(ctx)
	if err != nil {
		return
	}
	respBody := string(resp)
	// 执行url匹配正则
	urlRegex := regexp.MustCompile(enums2.SuspectURLRegex)
	urlList := urlRegex.FindAllString(respBody, -1)
	// 遍历全部url列表
	for _, url := range urlList {
		url = url[1 : len(url)-1]
		urlLower := strings.ToLower(url)
		if strings.HasPrefix(urlLower, "image/x-icon") || strings.HasPrefix(urlLower, "text/css") || strings.HasPrefix(urlLower, "text/javascript") {
			continue
		}
		tab.AddResultFormCustomUrl(enums2.GET, url, enums2.FromJSFile)
	}
	// 执行用户自定义正则
	for _, custom := range tab.config.CustomDefinedRegex {
		customRegex := regexp.MustCompile(custom)
		customList := customRegex.FindAllString(respBody, -1)
		tab.CustomDefinedRegexResultList = append(tab.CustomDefinedRegexResultList, struct {
			Regexp string
			Result []string
		}{Regexp: custom, Result: customList})
	}
}

// ParseRequestURLFromResponseHeader 解析请求的url从返回的响应头中
func (tab *Tab) ParseRequestURLFromResponseHeader(v *network.EventResponseReceived) {
	defer tab.WaitGroup.Done()
	// 这个响应头也许也会包含诸多的链接
	for key, _ := range v.Response.Headers {
		if key == "Link" || key == "Content-Location" || key == "Location" || key == "Refresh" {
			tab.AddResultFormCustomUrl(enums2.GET, v.Response.URL, enums2.FromHeader)
		}
	}
}

// GetContentCharset 从请求头中获取字符编码
func (tab *Tab) GetContentCharset(v *network.EventResponseReceived) {
	defer tab.WaitGroup.Done()
	var getCharsetRegex = regexp.MustCompile("charset=(.+)$")
	for key, value := range v.Response.Headers {
		if strings.Contains(strings.ToLower(key), strings.ToLower("Content-Type")) {
			value := value.(string)
			if strings.Contains(value, "charset") {
				value = getCharsetRegex.FindString(value)
				value = strings.ToUpper(strings.Replace(value, "charset=", "", -1))
				tab.PageCharset = value
				tab.PageCharset = strings.TrimSpace(tab.PageCharset)
			}
		}
	}
}

// HandleRedirectionResponse  控制重定向响应
func (tab *Tab) HandleRedirectionResponse(v *network.EventResponseReceivedExtraInfo) {
	defer tab.WaitGroup.Done()
	statusCode := tab.GetStatusCode(v.HeadersText)
	// 导航请求，且返回重定向
	if 300 <= statusCode && statusCode < 400 {
		tab.FoundRedirection = true
	}
}

// GetStatusCode  获取请求状态吗
func (tab *Tab) GetStatusCode(headerText string) int {
	rspInput := strings.NewReader(headerText)
	rspBuf := bufio.NewReader(rspInput)
	tp := textproto.NewReader(rspBuf)
	line, err := tp.ReadLine()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return 0
	}
	parts := strings.Split(line, " ")
	if len(parts) < 3 {
		return 0
	}
	code, _ := strconv.Atoi(parts[1])
	return code
}

// HandleAuthRequired 控制401的弹窗认证
func (tab *Tab) HandleAuthRequired(req *fetch.EventAuthRequired) {
	defer tab.WaitGroup.Done()
	ctx := tab.GetCDPExecutor()
	var authRes = fetch.AuthChallengeResponse{}
	if tab.config.Custom401Auth.Password != "" && tab.config.Custom401Auth.Username != "" {
		authRes = fetch.AuthChallengeResponse{
			Response: fetch.AuthChallengeResponseResponseProvideCredentials,
			Username: tab.config.Custom401Auth.Username,
			Password: tab.config.Custom401Auth.Password,
		}
	} else {
		authRes = fetch.AuthChallengeResponse{
			Response: fetch.AuthChallengeResponseResponseProvideCredentials,
			Username: "Admin",
			Password: "123456",
		}
	}
	//  做认证
	_ = fetch.ContinueWithAuth(req.RequestID, &authRes).Do(ctx)
}

func (tab *Tab) HandleBindingCalled(event *runtime.EventBindingCalled) {
	defer tab.WaitGroup.Done()
	payload := []byte(event.Payload)
	var bcPayload BindingCallPayload
	_ = json.Unmarshal(payload, &bcPayload)
	if bcPayload.Name == "addLink" && len(bcPayload.Args) > 1 {
		tab.AddResultFormCustomUrl(enums2.GET, bcPayload.Args[0], bcPayload.Args[1])
	}
	if bcPayload.Name == "Test" {
		fmt.Println(bcPayload.Args)
	}
	tab.Evaluate(fmt.Sprintf(enums2.DeliverResultJS, bcPayload.Name, bcPayload.Seq, "s"))
}
