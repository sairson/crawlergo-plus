package enums

import "time"

const (
	DefaultUA               = "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.0 Safari/537.36"
	MaxTabsCount            = 10
	TabRunTimeout           = 15 * time.Second
	DefaultInputText        = "Universe"
	FormInputKeyword        = "Universe"
	SuspectURLRegex         = `(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;|*()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml|do|cgi)(?:[\?|#][^"|']{0,}|)))(?:"|')`
	URLRegex                = `((https?|ftp|file):)?//[-A-Za-z0-9+&@#/%?=~_|!:,.;]+[-A-Za-z0-9+&@#/%=~_|]`
	AttrURLRegex            = ``
	DomContentLoadedTimeout = 5 * time.Second
	EventTriggerInterval    = 100 * time.Millisecond // 单位毫秒
	BeforeExitDelay         = 1 * time.Second
	DefaultEventTriggerMode = EventTriggerAsync
	MaxCrawlCount           = 300
)

// 事件触发模式
const (
	EventTriggerAsync = "async"
	EventTriggerSync  = "sync"
)

var DefaultInputTextMap = map[string]map[string]interface{}{
	"mail": {
		"keyword": []string{"mail"},
		"value":   "universe@gmail.com",
	},
	"code": {
		"keyword": []string{"yanzhengma", "code", "ver", "captcha"},
		"value":   "123a",
	},
	"phone": {
		"keyword": []string{"phone", "number", "tel", "shouji"},
		"value":   "18812345678",
	},
	"username": {
		"keyword": []string{"name", "user", "id", "login", "account"},
		"value":   "crawlergo@gmail.com",
	},
	"password": {
		"keyword": []string{"pass", "pwd"},
		"value":   "123456",
	},
	"qq": {
		"keyword": []string{"qq", "wechat", "tencent", "weixin"},
		"value":   "123456789",
	},
	"IDCard": {
		"keyword": []string{"card", "shenfen"},
		"value":   "511702197409284963",
	},
	"url": {
		"keyword": []string{"url", "site", "web", "blog", "link"},
		"value":   "https://universe.nice.cn/",
	},
	"date": {
		"keyword": []string{"date", "time", "year", "now"},
		"value":   "2023-01-01",
	},
	"number": {
		"keyword": []string{"day", "age", "num", "count"},
		"value":   "10",
	},
}

var DefaultIgnoreKeywords = []string{"logout", "quit", "exit"}

const DefaultFuzzDict = "11/123/2017/2018/message/mis/model/abstract/account/act/action" +
	"/activity/ad/address/ajax/alarm/api/app/ar/attachment/auth/authority/award/back/backup/bak/base" +
	"/bbs/bbs1/cms/bd/gallery/game/gift/gold/bg/bin/blacklist/blog/bootstrap/brand/build/cache/caches" +
	"/caching/cacti/cake/captcha/category/cdn/ch/check/city/class/classes/classic/client/cluster" +
	"/collection/comment/commit/common/commons/components/conf/config/mysite/confs/console/consumer" +
	"/content/control/controllers/lib/crontab/crud/css/daily/dashboard/data/database/db/default/demo" +
	"/dev/doc/download/duty/es/eva/examples/excel/export/ext/fe/feature/file/files/finance/flashchart" +
	"/follow/forum/frame/framework/ft/group/gss/hello/helper/helpers/history/home/hr/htdocs/html/hunter" +
	"/image/img11/import/improve/inc/include/includes/index/info/install/interface/item/jobconsume/jobs" +
	"/json/kindeditor/l/languages/lib/libraries/libs/link/lite/local/log/login/logs/mail/main" +
	"/maintenance/manage/manager/manufacturer/menus/models/modules/monitor/movie/mysql/n/nav/network" +
	"/news/notice/nw/oauth/other/page/pages/passport/pay/pcheck/people/person/php/phprpc" +
	"/phptest/picture/pl/platform/pm/portal/post/product/project/protected/proxy/ps/public/qq/question" +
	"/quote/redirect/redisclient/report/resource/resources/s/save/schedule/schema/script/scripts/search" +
	"/security/server/service/shell/show/simple/site/sites/skin/sms/soap/sola/sort/spider/sql/stat" +
	"/static/statistics/stats/submit/subways/survey/sv/syslog/system/tag/task/tasks/tcpdf/template" +
	"/templates/test/tests/ticket/tmp/token/tool/tools/top/tpl/txt/upload/uploadify/uploads/url/user" +
	"/util/v1/v2/vendor/view/views/web/weixin/widgets/wm/wordpress/workspace/ws/www/www2/wwwroot/zone" +
	"/admin/admin_bak/mobile/m/js"
