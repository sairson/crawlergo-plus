package internal

import (
	"encoding/json"
	"fmt"
	"github.com/sairson/crawlergo/internal/engine/enums"
	"github.com/sairson/crawlergo/internal/engine/httplib"
	"github.com/sairson/crawlergo/internal/engine/httplib/urllib"
	"github.com/sairson/crawlergo/internal/option"
	"testing"
)

func TestCrawler_Run(t *testing.T) {
	var targets []*httplib.RequestCrawler
	var urls = []string{"http://testphp.vulnweb.com/"}
	var postData string = "username=admin&password=password" // 提交的post数据
	// 默认忽略的关键字
	ignoreKeyWords := []string{"logout", "quit", "exit"}
	var defaultTaskOptions = option.TaskOptions{
		EventTriggerMode:        enums.EventTriggerAsync,       // 事件的出发方式
		EventTriggerInterval:    enums.EventTriggerInterval,    // 时间触发的间隔
		BeforeExitDelay:         enums.BeforeExitDelay,         // 退出前的等待时间，等待DOM渲染，等待XHR发出捕获
		NoHeadless:              false,                         // 无头模式
		DomContentLoadedTimeout: enums.DomContentLoadedTimeout, // dom节点的默认超时
		TabRunTimeout:           enums.TabRunTimeout,           // 单个tab页的运行超时
		EncodeURLWithCharset:    true,                          // 是否使用检测到的字符集自行编码
		Proxy:                   "",                            // 是否使用编码
		PathFormRobots:          true,                          // 是否使用robots来爬取
		PathFuzz:                false,                         // 是否使用路径fuzz
		PathFormSitemap:         true,
		MaxTabCount:             8,
		FilterMode:              "smart",
		Custom401Auth: struct {
			Username string
			Password string
		}{Username: "admin", Password: "admin"},
		MaxCrawlerCount:    enums.MaxCrawlCount,
		ExtraHeadersString: "",
		ChromiumPath:       "",
	}
	defaultTaskOptions.IgnoreKeywords = ignoreKeyWords

	for _, url := range urls {
		var req *httplib.RequestCrawler
		newUrl, err := urllib.GetURL(url)
		if err != nil {
			continue
		}
		if postData != "" {
			req = httplib.GetCrawlerRequest(enums.POST, newUrl, getOption(defaultTaskOptions, postData))
		} else {
			req = httplib.GetCrawlerRequest(enums.GET, newUrl, getOption(defaultTaskOptions, postData))
		}
		req.Proxy = defaultTaskOptions.Proxy
		targets = append(targets, req)
	}
	task, err := NewTabCrawlerGoTask(targets, defaultTaskOptions)
	task.ResultCallback = func(i *httplib.RequestCrawler) error {
		fmt.Println(i)
		return nil
	}
	if err != nil {
		return
	}
	// 开始爬虫
	task.Run()
}

func getOption(taskOptions option.TaskOptions, postData string) httplib.OptionsCrawler {
	var options httplib.OptionsCrawler
	if postData != "" {
		options.PostData = postData
	}
	if taskOptions.ExtraHeadersString != "" {
		err := json.Unmarshal([]byte(taskOptions.ExtraHeadersString), &taskOptions.ExtraHeaders)
		if err != nil {
			panic(err)
		}
		options.Headers = taskOptions.ExtraHeaders
	}
	return options
}
