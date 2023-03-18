package enums

import "regexp"

var SupportContentType = []string{JSON, URLENCODED}

const (
	JSON       = "application/json"
	URLENCODED = "application/x-www-form-urlencoded"
	MULTIPART  = "multipart/form-data"
)

const (
	FromTarget      = "Target"     //初始输入的目标
	FromNavigation  = "Navigation" //页面导航请求
	FromXHR         = "XHR"        //ajax异步请求
	FromDOM         = "DOM"        //dom解析出来的请求
	FromJSFile      = "JavaScript" //JS脚本中解析
	FromFuzz        = "PathFuzz"   //初始path fuzz
	FromRobots      = "robots.txt" //robots.txt
	FromSitemap     = "sitemap.xml"
	FromComment     = "Comment"     //页面中的注释
	FromWebSocket   = "WebSocket"   // websocket请求
	FromEventSource = "EventSource" // 事件源
	FromFetch       = "Fetch"
	FromHistoryAPI  = "HistoryAPI"
	FromOpenWindow  = "OpenWindow"
	FromHashChange  = "HashChange"
	FromStaticRes   = "StaticResource"
	FromStaticRegex = "StaticRegex"
	FromHeader      = "Header" // 响应头获取
)

// 请求方法
const (
	GET     = "GET"
	POST    = "POST"
	PUT     = "PUT"
	DELETE  = "DELETE"
	HEAD    = "HEAD"
	OPTIONS = "OPTIONS"
)

var ChineseRegex = regexp.MustCompile("[\u4e00-\u9fa5]+")
var UrlencodeRegex = regexp.MustCompile("(?:%[A-Fa-f0-9]{2,6})+")
var UnicodeRegex = regexp.MustCompile(`(?:\\u\w{4})+`)
var OnlyAlphaRegex = regexp.MustCompile("^[a-zA-Z]+$")
var OnlyAlphaUpperRegex = regexp.MustCompile("^[A-Z]+$")
var AlphaUpperRegex = regexp.MustCompile("[A-Z]+")
var AlphaLowerRegex = regexp.MustCompile("[a-z]+")
var ReplaceNumRegex = regexp.MustCompile(`[0-9]+\.[0-9]+|\d+`)
var OnlyNumberRegex = regexp.MustCompile(`^[0-9]+$`)
var NumberRegex = regexp.MustCompile(`[0-9]+`)
var OneNumberRegex = regexp.MustCompile(`[0-9]`)
var NumSymbolRegex = regexp.MustCompile(`\.|_|-`)
var TimeSymbolRegex = regexp.MustCompile(`-|:|\s`)
var OnlyAlphaNumRegex = regexp.MustCompile(`^[0-9a-zA-Z]+$`)
var MarkedStringRegex = regexp.MustCompile(`^{{.+}}$`)
var HtmlReplaceRegex = regexp.MustCompile(`\.shtml|\.html|\.htm`)

const (
	CustomValueMark    = "{{Custom}}"
	FixParamRepeatMark = "{{fix_param}}"
	FixPathMark        = "{{fix_path}}"
	TooLongMark        = "{{long}}"
	NumberMark         = "{{number}}"
	ChineseMark        = "{{chinese}}"
	UpperMark          = "{{upper}}"
	LowerMark          = "{{lower}}"
	UrlEncodeMark      = "{{urlencode}}"
	UnicodeMark        = "{{unicode}}"
	BoolMark           = "{{bool}}"
	ListMark           = "{{list}}"
	TimeMark           = "{{time}}"
	MixAlphaNumMark    = "{{mix_alpha_num}}"
	MixSymbolMark      = "{{mix_symbol}}"
	MixNumMark         = "{{mix_num}}"
	NoLowerAlphaMark   = "{{no_lower}}"
	MixStringMark      = "{{mix_str}}"
)

const (
	MaxParentPathCount         = 32 // 相对于上一级目录，本级path目录的数量修正最大值
	MaxParamKeySingleCount     = 8  // 某个URL参数名重复修正最大值
	MaxParamKeyAllCount        = 10 // 本轮所有URL中某个参数名的重复修正最大值
	MaxPathParamEmptyCount     = 10 // 某个path下的参数值为空，参数名个数修正最大值
	MaxPathParamKeySymbolCount = 5  // 某个Path下的某个参数的标记数量超过此值，则该参数被全局标记
)
