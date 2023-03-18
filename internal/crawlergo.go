package internal

import (
	"encoding/json"
	mapset "github.com/deckarep/golang-set"
	"github.com/panjf2000/ants/v2"
	engine2 "github.com/sairson/crawlergo/internal/engine"
	enums2 "github.com/sairson/crawlergo/internal/engine/enums"
	"github.com/sairson/crawlergo/internal/engine/httplib"
	"github.com/sairson/crawlergo/internal/expression"
	"github.com/sairson/crawlergo/internal/filter"
	"github.com/sairson/crawlergo/internal/option"
	"sync"
	"time"
)

type Crawler struct {
	Browser             *engine2.Browser
	RootDomain          string // 爬取的网站跟域名,主要用于子域名的收集
	Pool                *ants.Pool
	Targets             []*httplib.RequestCrawler
	WaitGroup           sync.WaitGroup
	Option              *option.TaskOptions
	SmartFilter         filter.SmartFilter                    // 过滤对象
	ResultCallback      func(i *httplib.RequestCrawler) error // 结果回调函数
	Result              CrawlerResult                         // 爬虫最终结果
	CrawlerAlreadyCount int                                   // 已经爬取过的总数
	CrawlerCountLock    sync.Mutex                            // 爬虫总数锁
}

type CrawlerResult struct {
	RequestList           []*httplib.RequestCrawler // 返回的同域名结果
	AllRequestList        []*httplib.RequestCrawler // 所有域名的请求
	AllDomainList         []string                  // 所有域名列表
	SubDomainList         []string                  // 子域名列表
	MergeResultAttachLock sync.Mutex                // 合并结果时的加锁
}

type TabCrawler struct {
	crawler *Crawler                // 爬虫
	browser *engine2.Browser        // 浏览器
	request *httplib.RequestCrawler // 请求
}

// NewTabCrawlerGoTask 新建一个tab页爬虫事件
func NewTabCrawlerGoTask(targets []*httplib.RequestCrawler, options option.TaskOptions) (*Crawler, error) {
	var crawler = &Crawler{
		Option: &options,
		SmartFilter: filter.SmartFilter{
			SimpleFilter: filter.SimpleFilter{},
		},
	}
	// 如果我们的目标数量 > 0
	if len(targets) > 0 {
		crawler.SmartFilter.SimpleFilter.HostLimit = targets[0].URL.Host
	}
	if len(targets) == 1 {
		_newReq := *targets[0]
		newReq := &_newReq
		_newURL := *_newReq.URL
		newReq.URL = &_newURL
		if targets[0].URL.Scheme == "http" {
			newReq.URL.Scheme = "https"
		} else {
			newReq.URL.Scheme = "http"
		}
		targets = append(targets, newReq)
	}
	crawler.Targets = targets[:]

	for _, req := range targets {
		req.Source = "Target"
	}
	//  执行一些函数来设置一些默认值
	for _, fn := range []option.TaskOptionOptFunc{
		crawler.WithTabRunTimeout(enums2.TabRunTimeout), // 带有默认值设置函数
		crawler.WithMaxTabsCount(enums2.MaxTabsCount),
		crawler.WithMaxCrawlCount(enums2.MaxCrawlCount),
		crawler.WithDomContentLoadedTimeout(enums2.DomContentLoadedTimeout),
		crawler.WithEventTriggerInterval(enums2.EventTriggerInterval),
		crawler.WithBeforeExitDelay(enums2.BeforeExitDelay),
		crawler.WithEventTriggerMode(enums2.DefaultEventTriggerMode),
		crawler.WithIgnoreKeywords(enums2.DefaultIgnoreKeywords),
	} {
		fn(&options)
	}
	// 初始化请求头字符串
	if options.ExtraHeadersString != "" {
		err := json.Unmarshal([]byte(options.ExtraHeadersString), &options.ExtraHeaders)
		if err != nil {
			return nil, err
		}
	}
	// 初始化浏览器
	crawler.Browser, _ = engine2.InitBrowser(options.ChromiumPath, options.ExtraHeaders, options.Proxy, options.NoHeadless)
	// 初始化我们的根域名
	crawler.RootDomain = targets[0].URL.RootDomain()
	// 智能过滤器初始化
	crawler.SmartFilter.Init()

	// 创建协程池
	p, _ := ants.NewPool(options.MaxTabCount)
	crawler.Pool = p

	return crawler, nil
}

// WithTabRunTimeout 设置每一个tab页的运行超时
func (crawler *Crawler) WithTabRunTimeout(gen time.Duration) option.TaskOptionOptFunc {
	return func(tc *option.TaskOptions) {
		if tc.TabRunTimeout == 0 {
			tc.TabRunTimeout = gen
		}
	}
}

// WithMaxCrawlCount 设置最大的爬取数量
func (crawler *Crawler) WithMaxCrawlCount(maxCrawlCount int) option.TaskOptionOptFunc {
	return func(tc *option.TaskOptions) {
		if tc.MaxCrawlerCount == 0 {
			tc.MaxCrawlerCount = maxCrawlCount
		}
	}
}

// WithMaxTabsCount 设置最大tab页的打开数量
func (crawler *Crawler) WithMaxTabsCount(gen int) option.TaskOptionOptFunc {
	return func(tc *option.TaskOptions) {
		if tc.MaxTabCount == 0 {
			tc.MaxTabCount = gen
		}
	}
}

// WithEventTriggerMode 设置页面事件的触发方式async,sync
func (crawler *Crawler) WithEventTriggerMode(gen string) option.TaskOptionOptFunc {
	return func(tc *option.TaskOptions) {
		if tc.EventTriggerMode == "" {
			tc.EventTriggerMode = gen
		}
	}
}

// WithEventTriggerInterval 设置页面的触发间隔
func (crawler *Crawler) WithEventTriggerInterval(gen time.Duration) option.TaskOptionOptFunc {
	return func(tc *option.TaskOptions) {
		if tc.EventTriggerInterval == 0 {
			tc.EventTriggerInterval = gen
		}
	}
}

// WithBeforeExitDelay 设置退出前的等待间隔
func (crawler *Crawler) WithBeforeExitDelay(gen time.Duration) option.TaskOptionOptFunc {
	return func(tc *option.TaskOptions) {
		if tc.BeforeExitDelay == 0 {
			tc.BeforeExitDelay = gen
		}
	}
}

// WithIgnoreKeywords 设置需要忽略的关键字
func (crawler *Crawler) WithIgnoreKeywords(gen []string) option.TaskOptionOptFunc {
	return func(tc *option.TaskOptions) {
		if tc.IgnoreKeywords == nil || len(tc.IgnoreKeywords) == 0 {
			tc.IgnoreKeywords = gen
		}
	}
}

// WithDomContentLoadedTimeout 设置dom加载的超时
func (crawler *Crawler) WithDomContentLoadedTimeout(gen time.Duration) option.TaskOptionOptFunc {
	return func(tc *option.TaskOptions) {
		if tc.DomContentLoadedTimeout == 0 {
			tc.DomContentLoadedTimeout = gen
		}
	}
}

func (crawler *Crawler) Run() {
	defer crawler.Pool.Release()                // 释放爬虫使用的协程池
	defer crawler.Browser.CloseTabsAndBrowser() // 关闭浏览器的所有标签页和自身

	// 新建一个表达式处理
	crawlerExpression := new(expression.CrawlerExpression)
	// 从robots.txt中获取
	if crawler.Option.PathFormRobots {
		if result, _ := crawlerExpression.Robots(*crawler.Targets[0], crawler.ResultCallback); len(result) > 0 {
			crawler.Targets = append(crawler.Targets, result...)
		}
	}
	if crawler.Option.PathFormSitemap {
		if result, _ := crawlerExpression.Sitemap(*crawler.Targets[0], crawler.ResultCallback); len(result) > 0 {
			crawler.Targets = append(crawler.Targets, result...)
		}
	}
	if crawler.Option.PathFuzz && crawler.Option.FuzzDictPath != "" {
		if result, _ := crawlerExpression.DoFuzzFromCustomDict(*crawler.Targets[0], crawler.ResultCallback, crawler.Option.FuzzDictPath); len(result) > 0 {
			crawler.Targets = append(crawler.Targets, result...)
		}
	} else if crawler.Option.PathFuzz {
		if result, _ := crawlerExpression.DoFuzzFromDefaultDict(*crawler.Targets[0], crawler.ResultCallback); len(result) > 0 {
			crawler.Targets = append(crawler.Targets, result...)
		}
	}
	crawler.Result.AllRequestList = crawler.Targets[:]

	// 执行tab任务做深度的自动化爬虫
	var initDeepCrawler []*httplib.RequestCrawler
	for i := 0; i < len(crawler.Targets); i++ {
		if crawler.SmartFilter.DoFilter(crawler.Targets[i]) {
			continue
		}
		initDeepCrawler = append(initDeepCrawler, crawler.Targets[i])
		crawler.Result.RequestList = append(crawler.Result.RequestList, crawler.Targets[i])
	}

	// 执行更深层的tab页爬虫
	for i := 0; i < len(initDeepCrawler); i++ {
		if !engine2.IsIgnoredByKeywordMatch(*initDeepCrawler[i], crawler.Option.IgnoreKeywords) {
			crawler.DeepCrawlerTaskPool(initDeepCrawler[i])
		}
	}
	crawler.WaitGroup.Wait()

	// 对全部请求进行唯一去重

	todoFilterAll := make([]*httplib.RequestCrawler, len(crawler.Result.AllRequestList))
	copy(todoFilterAll, crawler.Result.AllRequestList)

	crawler.Result.AllRequestList = []*httplib.RequestCrawler{}
	var simpleFilter filter.SimpleFilter
	for _, req := range todoFilterAll {
		if !simpleFilter.UniqueFilter(req) {
			crawler.Result.AllRequestList = append(crawler.Result.AllRequestList, req)
		}
	}

	// 我们执行前返回全部的域名调用
	// 全部域名
	var domainCollect = new(DomainCollect)
	crawler.Result.AllDomainList = domainCollect.AllDomainCollect(crawler.Result.AllRequestList)
	// 子域名
	crawler.Result.SubDomainList = domainCollect.SubDomainCollect(crawler.Result.AllRequestList, crawler.RootDomain)
}

// DeepCrawlerTaskPool 深度的爬虫任务，主要通过tab标签页任务，来进行爬取
func (crawler *Crawler) DeepCrawlerTaskPool(req *httplib.RequestCrawler) {
	crawler.CrawlerCountLock.Lock()
	// 如果爬取的总数已经大于最大的爬取数量后
	if crawler.CrawlerAlreadyCount >= crawler.Option.MaxCrawlerCount {
		crawler.CrawlerCountLock.Unlock()
		return
	} else {
		crawler.CrawlerAlreadyCount += 1
	}
	crawler.CrawlerCountLock.Unlock()
	crawler.WaitGroup.Add(1)
	tabCrawler := &TabCrawler{crawler: crawler, browser: crawler.Browser, request: req}
	go func() {
		err := crawler.Pool.Submit(tabCrawler.TabCrawlerTask)
		if err != nil {
			crawler.WaitGroup.Done()
		}
	}()
}

func (t *TabCrawler) TabCrawlerTask() {
	defer t.crawler.WaitGroup.Done()
	tab := engine2.NewCrawlerTab(t.browser, *t.request, engine2.TabConfig{
		TabRunTimeout:           t.crawler.Option.TabRunTimeout,
		DomContentLoadedTimeout: t.crawler.Option.DomContentLoadedTimeout,
		EventTriggerMode:        t.crawler.Option.EventTriggerMode,
		EventTriggerInterval:    t.crawler.Option.EventTriggerInterval,
		BeforeExitDelay:         t.crawler.Option.BeforeExitDelay,
		EncodeURLWithCharset:    t.crawler.Option.EncodeURLWithCharset,
		IgnoreKeywords:          t.crawler.Option.IgnoreKeywords,
		CustomFormValues:        t.crawler.Option.CustomFormValues,
		CustomFormKeywordValues: t.crawler.Option.CustomFormKeywordValues,
		Custom401Auth:           t.crawler.Option.Custom401Auth,
		RootDomain:              t.crawler.RootDomain,
	})
	tab.HrefClick = mapset.NewSet()               // 链接是否点击过了
	tab.ResultCallback = t.crawler.ResultCallback // 回调函数必须有
	tab.Start()
	// 收集全部的tab页结果
	t.crawler.Result.MergeResultAttachLock.Lock()
	t.crawler.Result.AllRequestList = append(t.crawler.Result.AllRequestList, tab.ResultList...)
	t.crawler.Result.MergeResultAttachLock.Unlock()

	// 智能过滤全部结果

	for _, req := range tab.ResultList {
		if t.crawler.Option.FilterMode == enums2.SimpleFilterMode {
			if !t.crawler.SmartFilter.SimpleFilter.DoFilter(req) {
				t.crawler.Result.MergeResultAttachLock.Lock()
				t.crawler.Result.RequestList = append(t.crawler.Result.RequestList, req)
				t.crawler.Result.MergeResultAttachLock.Unlock()
				if !engine2.IsIgnoredByKeywordMatch(*req, t.crawler.Option.IgnoreKeywords) {
					t.crawler.DeepCrawlerTaskPool(req)
				}
			}
		} else {
			if !t.crawler.SmartFilter.DoFilter(req) {
				t.crawler.Result.MergeResultAttachLock.Lock()
				t.crawler.Result.RequestList = append(t.crawler.Result.RequestList, req)
				t.crawler.Result.MergeResultAttachLock.Unlock()
				if !engine2.IsIgnoredByKeywordMatch(*req, t.crawler.Option.IgnoreKeywords) {
					t.crawler.DeepCrawlerTaskPool(req)
				}
			}
		}
	}
}
