package engine

import (
	"context"
	browserdp "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
	"sync"
	"time"
)

type Browser struct {
	Context      *context.Context
	Cancel       *context.CancelFunc
	Tabs         []*Tabs
	ExtraHeaders map[string]interface{}
	Mutex        sync.Mutex
}

type Tabs struct {
	TabContext *context.Context
	TabCancel  *context.CancelFunc
}

func InitBrowser(chromium string, extraHeaders map[string]interface{}, proxy string, noHeadless bool) (*Browser, error) {
	var browser = &Browser{}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// 是否启用无头模式
		chromedp.Flag("headless", !noHeadless),
		// 禁用GPU,不显示GUI
		chromedp.Flag("disable-gpu", true),
		// 取消沙盒模式
		chromedp.Flag("no-sandbox", true),
		// 忽略证书验证
		chromedp.Flag("ignore-certificate-errors", true),

		chromedp.Flag("disable-images", true),

		chromedp.Flag("disable-web-security", true),

		chromedp.Flag("disable-xss-auditor", true),

		chromedp.Flag("disable-setuid-sandbox", true),

		chromedp.Flag("allow-running-insecure-content", true),

		chromedp.Flag("disable-webgl", true),

		chromedp.Flag("disable-popup-blocking", true),

		chromedp.WindowSize(1920, 1080),
	)
	if proxy != "" {
		opts = append(opts, chromedp.ProxyServer(proxy))
	}
	if chromium != "" && len(chromium) > 0 {
		// 指定二进制程序执行路径
		opts = append(opts, chromedp.ExecPath(chromium))
	}
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	browserCtx, _ := chromedp.NewContext(allocCtx) //chromedp.WithLogf(log.Printf),
	// 如果需要在一个浏览器上创建多个tab，则需要先创建浏览器的上下文，即运行下面的语句
	err := chromedp.Run(browserCtx)
	if err != nil {
		return nil, err
	}
	browser.Cancel = &cancel
	browser.Context = &browserCtx
	browser.ExtraHeaders = extraHeaders
	return browser, nil
}

// NewTab 新建一个Tab页
func (browser *Browser) NewTab(timeout time.Duration) (*context.Context, context.CancelFunc) {
	// 添加锁
	browser.Mutex.Lock()
	ctx, cancel := chromedp.NewContext(*browser.Context)
	tCtx, cancel2 := context.WithTimeout(ctx, timeout)
	_ = cancel2 // 用不上
	// 我们每一个tab页都集中管理并返回
	browser.Tabs = append(browser.Tabs, &Tabs{
		TabContext: &tCtx,
		TabCancel:  &cancel,
	})
	// 新建页解锁
	browser.Mutex.Unlock()
	return &tCtx, cancel
}

// CloseTabsAndBrowser 关闭相关的全部标签
func (browser *Browser) CloseTabsAndBrowser() {
	// 关闭tab页
	for _, tab := range browser.Tabs {
		(*tab.TabCancel)()
		err := browserdp.Close().Do(*tab.TabContext)
		if err != nil {
			continue
		}
	}
	// 关闭最终的浏览器
	err := browserdp.Close().Do(*browser.Context)
	defer (*browser.Cancel)()
	if err != nil {
		return
	}
}
