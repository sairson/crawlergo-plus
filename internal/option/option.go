package option

import (
	mapset "github.com/deckarep/golang-set"
	"time"
)

var (
	// StaticSuffix 静态资源后缀
	StaticSuffix = []string{
		"png", "gif", "jpg", "mp4", "mp3", "mng", "pct", "bmp", "jpeg", "pst", "psp", "ttf",
		"tif", "tiff", "ai", "drw", "wma", "ogg", "wav", "ra", "aac", "mid", "au", "aiff",
		"dxf", "eps", "ps", "svg", "3gp", "asf", "asx", "avi", "mov", "mpg", "qt", "rm",
		"wmv", "m4a", "bin", "xls", "xlsx", "ppt", "pptx", "doc", "docx", "odt", "ods", "odg",
		"odp", "exe", "zip", "rar", "tar", "gz", "iso", "rss", "pdf", "txt", "dll", "ico",
		"gz2", "apk", "crt", "woff", "map", "woff2", "webp", "less", "dmg", "bz2", "otf", "swf",
		"flv", "mpeg", "dat", "xsl", "csv", "cab", "exif", "wps", "m4v", "rmvb",
	}
	StaticSuffixSet mapset.Set
)

var (
	ScriptSuffix = []string{
		"php", "asp", "jsp", "asa", "action", "do",
	}
	ScriptSuffixSet mapset.Set
)

func init() {
	StaticSuffixSet = initSet(StaticSuffix) // 初始化静态资源后缀
	ScriptSuffixSet = initSet(ScriptSuffix) // 初始化脚本资源后缀
}

func initSet(suffix []string) mapset.Set {
	set := mapset.NewSet()
	for _, s := range suffix {
		set.Add(s)
	}
	return set
}

type TaskOptions struct {
	MaxCrawlerCount         int                    // 最大爬取的数量
	FilterMode              string                 // 过滤模式,支持simple(普通),smart(智能),strict(严格)
	ExtraHeaders            map[string]interface{} // 额外的请求头
	ExtraHeadersString      string                 // 额外请求头字符串
	AllDomainReturn         bool                   // 全部域名收集
	SubDomainReturn         bool                   // 子域名收集
	NoHeadless              bool                   // chromedp的无头模式
	DomContentLoadedTimeout time.Duration          // dom节点加载超时
	TabRunTimeout           time.Duration          // 单个tab页打开超时
	PathFuzz                bool                   // 是否通过字典进行路径fuzz
	FuzzDictPath            string                 // Fuzz目录字典
	PathFormRobots          bool                   // 解析Robots文件找出路径
	PathFormSitemap         bool                   // 解析网站地图找出路径
	MaxTabCount             int                    // 允许开启的最大标签页数量,即同时爬取的数量
	ChromiumPath            string                 // chromium程序的启动路径
	EventTriggerMode        string                 // 事件触发的调用方式： 异步 或 顺序
	EventTriggerInterval    time.Duration          // 事件触发的间隔
	BeforeExitDelay         time.Duration          // 退出前的等待时间，等待DOM渲染，等待XHR发出捕获
	EncodeURLWithCharset    bool                   // 使用检测到的字符集自动编码URL
	IgnoreKeywords          []string               // 忽略的关键字，匹配上之后将不再扫描且不发送请求
	Proxy                   string                 // 请求代理
	CustomFormValues        map[string]string      // 自定义表单填充参数
	CustomFormKeywordValues map[string]string      // 自定义表单关键词填充内容
	CustomDefinedRegex      []string               // 用户自定义正则,这个正则会在获取到js,css,json等文件被发现时被执行
	Custom401Auth           struct {               // 用户自定义401认证
		Username string
		Password string
	}
}

type TaskOptionOptFunc func(*TaskOptions)
