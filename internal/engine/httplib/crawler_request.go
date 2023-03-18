package httplib

import (
	"encoding/json"
	"errors"
	"github.com/sairson/crawlergo/internal/engine/enums"
	"github.com/sairson/crawlergo/internal/engine/httplib/urllib"
	"github.com/sairson/crawlergo/pkg/utils"
	"net/url"
	"strings"
)

// OptionsCrawler 爬虫需要的option参数,主要记录请求头和提交的post数据
type OptionsCrawler struct {
	Headers  map[string]interface{}
	PostData string
}

type RequestCrawler struct {
	URL         *urllib.URL            // url地址
	Method      string                 // 请求方法
	Headers     map[string]interface{} // 请求头
	PostData    string                 // post提交的数据
	Filter      Filter                 // 过滤器
	Source      string                 // 请求源
	Redirection bool                   // 重定向标志
	Proxy       string                 // 代理
}

type Filter struct {
	PostDataId        string
	MarkedPath        string
	FragmentID        string
	PathId            string
	UniqueId          string
	MarkedQueryMap    map[string]interface{}
	QueryKeysId       string
	QueryMapId        string
	MarkedPostDataMap map[string]interface{}
}

func GetCrawlerRequest(method string, URL *urllib.URL, options ...OptionsCrawler) *RequestCrawler {
	var crawler = new(RequestCrawler)
	crawler.URL = URL
	crawler.Method = strings.ToUpper(method)
	if len(options) > 0 {
		// 我们只取最后一个配置信息
		for _, option := range options {
			if option.Headers != nil {
				crawler.Headers = option.Headers
			}
			if option.PostData != "" {
				crawler.PostData = option.PostData
			}
		}
	} else {
		crawler.Headers = make(map[string]interface{})
	}
	return crawler
}

// ContentType 获取请求的Content-Type
func (req *RequestCrawler) ContentType() (string, error) {
	var contentType string
	if ct, ok := req.Headers["Content-Type"]; ok {
		contentType = ct.(string)
	} else if ct, ok := req.Headers["Content-type"]; ok {
		contentType = ct.(string)
	} else if ct, ok := req.Headers["content-type"]; ok {
		contentType = ct.(string)
	} else {
		return "", errors.New("no content-type")
	}

	// 我们判断是否是我们支持的content-Type
	for _, ct := range enums.SupportContentType {
		if strings.HasPrefix(contentType, ct) {
			return contentType, nil
		}
	}
	return "", errors.New("dont support such content-type:" + contentType)
}

// CrawlerPostData 获取爬虫的post请求数据
func (req *RequestCrawler) CrawlerPostData() map[string]interface{} {
	contentType, err := req.ContentType()
	if err != nil {
		return map[string]interface{}{
			"key": req.PostData,
		}
	}
	if strings.HasPrefix(contentType, enums.JSON) {
		var result map[string]interface{}
		err = json.Unmarshal([]byte(req.PostData), &result)
		if err != nil {
			return map[string]interface{}{
				"key": req.PostData,
			}
		} else {
			return result
		}
	} else if strings.HasPrefix(contentType, enums.URLENCODED) {
		var result = map[string]interface{}{}
		r, err := url.ParseQuery(req.PostData)
		if err != nil {
			return map[string]interface{}{
				"key": req.PostData,
			}
		} else {
			for key, value := range r {
				if len(value) == 1 {
					result[key] = value[0]
				} else {
					result[key] = value
				}
			}
			return result
		}
	} else {
		return map[string]interface{}{
			"key": req.PostData,
		}
	}
}

func (req *RequestCrawler) UniqueId() string {
	if req.Redirection {
		return utils.StrToMd5(req.NoHeaderId() + "Redirection")
	} else {
		return req.NoHeaderId()
	}
}

func (req *RequestCrawler) NoHeaderId() string {
	return utils.StrToMd5(req.Method + req.URL.String() + req.PostData)
}
