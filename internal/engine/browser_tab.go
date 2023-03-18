package engine

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	mapset "github.com/deckarep/golang-set"
	"github.com/gogf/gf/encoding/gcharset"
	enums2 "github.com/sairson/crawlergo/internal/engine/enums"
	"github.com/sairson/crawlergo/internal/engine/httplib"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Tab struct {
	RootDomain                   string           // 爬虫根域名
	Context                      *context.Context // 页面上下文
	Cancel                       context.CancelFunc
	NavigateRequest              httplib.RequestCrawler    // 活跃的请求信息
	ExtraHeaders                 map[string]interface{}    // 额外的请求头
	ResultList                   []*httplib.RequestCrawler // tab爬虫结果列表
	ResultCallback               func(v *httplib.RequestCrawler) error
	CustomDefinedRegexResultList []struct {
		Regexp string
		Result []string
	}
	TopFrameId       string
	LoaderID         string
	NavNetworkID     string
	PageCharset      string
	PageBindings     map[string]interface{}
	FoundRedirection bool
	DocBodyNodeId    cdp.NodeID
	Lock             sync.Mutex
	config           TabConfig

	WaitGroup            sync.WaitGroup // 当前Tab页的等待同步计数
	CollectLinkWaitGroup sync.WaitGroup // 收集链接等待计数
	LoadedWaitGroup      sync.WaitGroup // Loaded之后的等待计数
	FormSubmitWaitGroup  sync.WaitGroup // 表单提交完毕的等待计数
	RemoveList           sync.WaitGroup // 移除事件监听
	DomWaitGroup         sync.WaitGroup // DOMContentLoaded 的等待计数
	FillFormWaitGroup    sync.WaitGroup // 填充表单任务
	HrefClick            mapset.Set
}

// TabConfig 每一个页面的配置信息
type TabConfig struct {
	TabRunTimeout           time.Duration
	DomContentLoadedTimeout time.Duration
	EventTriggerMode        string        // 事件触发的调用方式： 异步 或 顺序
	EventTriggerInterval    time.Duration // 事件触发的间隔 单位毫秒
	BeforeExitDelay         time.Duration // 退出前的等待时间，等待DOM渲染，等待XHR发出捕获
	EncodeURLWithCharset    bool
	IgnoreKeywords          []string // 忽略的关键字
	IgnoreStatic            bool     // 是否忽略静态资源文件
	CustomDefinedRegex      []string // 用户自定义正则,这个正则会在获取到js,css,json等文件被执行
	Custom401Auth           struct {
		Username string
		Password string
	}
	Proxy                   string
	CustomFormValues        map[string]string
	CustomFormKeywordValues map[string]string
	RootDomain              string
}

type BindingCallPayload struct {
	Name string   `json:"name"`
	Seq  int      `json:"seq"`
	Args []string `json:"args"`
}

func NewCrawlerTab(browser *Browser, navigateRequest httplib.RequestCrawler, config TabConfig) *Tab {
	// 先初始化
	var tab Tab
	tab.ExtraHeaders = make(map[string]interface{})
	var DomContentLoadedRun = false
	// 我们通过浏览器建立一个tab页
	tab.Context, tab.Cancel = browser.NewTab(config.TabRunTimeout)
	for key, value := range browser.ExtraHeaders {
		if !strings.Contains(key, "Host") {
			tab.ExtraHeaders[key] = value
		}
		navigateRequest.Headers[key] = value
	}
	tab.NavigateRequest = navigateRequest
	tab.config = config
	tab.DocBodyNodeId = 0
	// tab页初始配置完成,我们设置chromedp的监听tab页的上下文
	chromedp.ListenTarget(*tab.Context, func(ev interface{}) {
		switch v := ev.(type) {
		case *network.EventRequestWillBeSent: // 当发送http请求的时候
			if v.RequestID.String() == v.LoaderID.String() && strings.Contains(v.Type.String(), "Document") && tab.TopFrameId == "" {
				tab.LoaderID = v.LoaderID.String()
				tab.TopFrameId = v.FrameID.String()
			}
		case *fetch.EventRequestPaused: // 请求暂停，也就是请求拦截
			tab.WaitGroup.Add(1)
			go tab.InterceptTabRequest(v)

		case *network.EventResponseReceived: // 当请求被接收的时候
			// 我们需要解析全部的JS文件并找到请求,此时我们也可以匹配一些正则来获取密钥结果,当然还有css文件,当中也有可能有一些相关的url链接
			if strings.Contains(strings.ToLower(v.Response.MimeType), "text/css") || strings.Contains(strings.ToLower(v.Response.MimeType), "application/javascript") || strings.Contains(strings.ToLower(v.Response.MimeType), "text/html") || strings.ToLower(v.Response.MimeType) == "application/json" {
				tab.WaitGroup.Add(1)
				go tab.ParseRequestURLFormResponseText(v)
			}
			if v.RequestID.String() == tab.NavNetworkID {
				tab.WaitGroup.Add(1)
				go tab.GetContentCharset(v)
			}
			// 这里我们不单单要解析全部的JS文件,还要从响应头中获取一些相关信息
			if v.RequestID.String() == tab.NavNetworkID {
				tab.WaitGroup.Add(1)
				go tab.ParseRequestURLFromResponseHeader(v)
			}
		case *network.EventResponseReceivedExtraInfo: // 后端重定向请求
			if v.RequestID.String() == tab.NavNetworkID {
				tab.WaitGroup.Add(1)
				go tab.HandleRedirectionResponse(v)
			}
		case *fetch.EventAuthRequired: // 控制401认证,407认证
			tab.WaitGroup.Add(1)
			go tab.HandleAuthRequired(v)
		case *page.EventDomContentEventFired: // dom节点请求
			// 如果dom已经加载完成并运行
			if DomContentLoadedRun {
				return
			}
			DomContentLoadedRun = true
			tab.WaitGroup.Add(1)
			go tab.AfterDOMLoadedToRunClickAndJavaScript()
		case *page.EventLoadEventFired:
			// 如果dom已经加载完成并运行
			if DomContentLoadedRun {
				return
			}
			DomContentLoadedRun = true
			tab.WaitGroup.Add(1)
			go tab.AfterDOMLoadedToRunClickAndJavaScript()
		case *page.EventJavascriptDialogOpening:
			tab.WaitGroup.Add(1)
			go tab.DismissDialog()
		case *runtime.EventBindingCalled: // 控制暴漏的函数
			tab.WaitGroup.Add(1)
			go tab.HandleBindingCalled(v)
		}
	})
	return &tab
}

// IsIgnoredByKeywordMatch 判断是否包含我们需要忽略的关键字
func IsIgnoredByKeywordMatch(req httplib.RequestCrawler, IgnoreKeywords []string) bool {
	for _, _str := range IgnoreKeywords {
		if strings.Contains(req.URL.String(), _str) {
			return true
		}
	}
	return false
}

// Start 开始执行爬虫任务
func (tab *Tab) Start() {
	defer tab.Cancel()
	if err := chromedp.Run(*tab.Context,
		RunWithTimeOut(tab.Context, tab.config.DomContentLoadedTimeout, chromedp.Tasks{
			runtime.Enable(),
			// 开启网络层API
			network.Enable(),
			// 开启请求拦截API
			fetch.Enable().WithHandleAuthRequests(true),
			// 添加回调函数绑定
			// XSS-Scan 使用的回调
			runtime.AddBinding("addLink"),
			runtime.AddBinding("Test"),
			// 初始化执行JS
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				_, err = page.AddScriptToEvaluateOnNewDocument(enums2.TabInitJS).Do(ctx)
				if err != nil {
					return err
				}
				return nil
			}),
			network.SetExtraHTTPHeaders(tab.ExtraHeaders),
			// 执行导航
			chromedp.Navigate(tab.NavigateRequest.URL.String()),
		}),
	); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
	}
	waitDone := func() <-chan struct{} {
		tab.WaitGroup.Wait()
		ch := make(chan struct{})
		defer close(ch)
		return ch
	}

	select {
	case <-waitDone():
	case <-time.After(tab.config.DomContentLoadedTimeout + time.Second*10):
	}
	// 等待收集全部的链接
	tab.CollectLinkWaitGroup.Add(3)
	go tab.CollectTabLinks() //收集全部的链接
	tab.CollectLinkWaitGroup.Wait()

	// 识别页面编码 并编码所有URL
	if tab.config.EncodeURLWithCharset {
		tab.DetectCharset()
		tab.EncodeAllURLWithCharset()
	}
}

// RunWithTimeOut 运行带有超时函数
func RunWithTimeOut(ctx *context.Context, timeout time.Duration, tasks chromedp.Tasks) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		_ = cancel
		return tasks.Do(timeoutContext)
	}
}

// CollectTabLinks 收集最后的全部链接
func (tab *Tab) CollectTabLinks() {
	go tab.CollectAttributeLinksFromTab() // 从属性中获取标签
	go tab.CollectObjectLinksFormTab()    // 从对象中获取标签
	go tab.CollectCommentLinksFromTab()   // 从注释中获取标签
}

// CollectAttributeLinksFromTab 收集全部的href链接
func (tab *Tab) CollectAttributeLinksFromTab() {
	defer tab.CollectLinkWaitGroup.Done()
	ctx := tab.GetCDPExecutor()
	// 收集 src href data-url action data等属性值属性值(有些是生成chat gpt生成完善的)
	attrNameList := []string{"src", "href", "link", "data-url", "codebase", "data-href", "action", "dynsrc", "image-href", "script-href", "data", "poster", "manifest", "ping", "longdesc", "usemap", "background", "source", "formaction"}
	for _, attrName := range attrNameList {
		tCtx, cancel := context.WithTimeout(ctx, time.Second*1)
		var attrs []map[string]string
		_ = chromedp.AttributesAll(fmt.Sprintf(`[%s]`, attrName), &attrs, chromedp.ByQueryAll).Do(tCtx)
		cancel()
		for _, attrMap := range attrs {
			tab.AddResultFormCustomUrl(enums2.GET, attrMap[attrName], enums2.FromDOM)
		}
	}
}

// CollectObjectLinksFormTab 收集对象中的链接
func (tab *Tab) CollectObjectLinksFormTab() {
	defer tab.CollectLinkWaitGroup.Done()
	ctx := tab.GetCDPExecutor()
	// 收集 object[data] links
	tCtx, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	var attrs []map[string]string
	_ = chromedp.AttributesAll(`object[data]`, &attrs, chromedp.ByQueryAll).Do(tCtx)
	for _, attrMap := range attrs {
		tab.AddResultFormCustomUrl(enums2.GET, attrMap["data"], enums2.FromDOM)
	}
}

// CollectCommentLinksFromTab 收集注释中的链接
func (tab *Tab) CollectCommentLinksFromTab() {
	defer tab.CollectLinkWaitGroup.Done()
	ctx := tab.GetCDPExecutor()
	// 收集注释中的链接
	var nodes []*cdp.Node
	tCtxComment, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	commentErr := chromedp.Nodes(`//comment()`, &nodes, chromedp.BySearch).Do(tCtxComment)
	if commentErr != nil {
		return
	}
	urlRegex := regexp.MustCompile(enums2.URLRegex)
	for _, node := range nodes {
		content := node.NodeValue
		urlList := urlRegex.FindAllString(content, -1)
		for _, url := range urlList {
			tab.AddResultFormCustomUrl(enums2.GET, url, enums2.FromComment)
		}
	}
}

// DetectCharset 做类型编码
func (tab *Tab) DetectCharset() {
	ctx := tab.GetCDPExecutor()
	tCtx, cancel := context.WithTimeout(ctx, time.Millisecond*500)
	defer cancel()
	var content string
	var ok bool
	var getCharsetRegex = regexp.MustCompile("charset=(.+)$")
	err := chromedp.AttributeValue(`meta[http-equiv=Content-Type]`, "content", &content, &ok, chromedp.ByQuery).Do(tCtx)
	if err != nil || !ok {
		return
	}
	if strings.Contains(content, "charset=") {
		charset := getCharsetRegex.FindString(content)
		if charset != "" {
			tab.PageCharset = strings.ToUpper(strings.Replace(charset, "charset=", "", -1))
			tab.PageCharset = strings.TrimSpace(tab.PageCharset)
		}
	}
}

// EncodeAllURLWithCharset 编码全部的url
func (tab *Tab) EncodeAllURLWithCharset() {
	if tab.PageCharset == "" || tab.PageCharset == "UTF-8" {
		return
	}
	for _, req := range tab.ResultList {
		newRawQuery, err := gcharset.UTF8To(tab.PageCharset, req.URL.RawQuery)
		if err == nil {
			req.URL.RawQuery = newRawQuery
		}
		newRawPath, err := gcharset.UTF8To(tab.PageCharset, req.URL.RawPath)
		if err == nil {
			req.URL.RawPath = newRawPath
		}
	}
}
