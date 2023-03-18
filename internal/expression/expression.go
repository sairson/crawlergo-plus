package expression

import (
	"encoding/xml"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	enums2 "github.com/sairson/crawlergo/internal/engine/enums"
	"github.com/sairson/crawlergo/internal/engine/httplib"
	"github.com/sairson/crawlergo/internal/engine/httplib/urllib"
	"github.com/sairson/crawlergo/internal/engine/requests"
	"github.com/sairson/crawlergo/pkg/utils"
	"regexp"
	"strings"
)

// 从一些其他的东西中获取请求
// 1.robots.txt
// 2.katana中得知还有sitemap.xml网站缩影

// Robots 从robots.txt中获取
func (expression *CrawlerExpression) Robots(navRequest httplib.RequestCrawler, callback func(i *httplib.RequestCrawler) error) ([]*httplib.RequestCrawler, error) {
	var result []*httplib.RequestCrawler
	url := strings.TrimSuffix(navRequest.URL.NoQueryUrl(), "/")
	requestURL := fmt.Sprintf("%s/robots.txt", url)
	resp, err := requests.Get(requestURL, utils.ConvertHeaders(navRequest.Headers), &requests.RequestOptions{
		AllowRedirect: false,
		Timeout:       5,
		Proxy:         navRequest.Proxy,
	})
	if err != nil {
		return result, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, errors.Wrap(err, fmt.Sprintf("status code is %v", resp.StatusCode))
	}
	// 我们查找url列表
	for _, v := range regexp.MustCompile(`(?:Disallow|Allow):.*?(/.+)`).FindAllString(resp.ToText(), -1) {
		url, err := urllib.GetURL(regexp.MustCompile(`(/.+)`).FindString(strings.TrimSpace(v)), *navRequest.URL)
		if err != nil {
			continue
		}
		request := httplib.GetCrawlerRequest(enums2.GET, url)
		request.Source = enums2.FromRobots
		_ = callback(request)
		result = append(result, request)
	}
	return result, nil
}

// Sitemap 从sitemap.xml网站地图中获取请求
func (expression *CrawlerExpression) Sitemap(navRequest httplib.RequestCrawler, callback func(i *httplib.RequestCrawler) error) ([]*httplib.RequestCrawler, error) {
	var result []*httplib.RequestCrawler
	url := strings.TrimSuffix(navRequest.URL.NoQueryUrl(), "/")
	requestURL := fmt.Sprintf("%s/sitemap.xml", url)
	resp, err := requests.Get(requestURL, utils.ConvertHeaders(navRequest.Headers), &requests.RequestOptions{
		AllowRedirect: false,
		Timeout:       5,
		Proxy:         navRequest.Proxy,
	})
	if err != nil {
		return result, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, errors.Wrap(err, fmt.Sprintf("status code is %v", resp.StatusCode))
	}
	sitemap := Sitemap{}
	if err := xml.NewDecoder(strings.NewReader(resp.ToText())).Decode(&sitemap); err != nil {
		return result, errors.Wrap(err, "could not decode xml")
	}
	for _, v := range sitemap.URLs {
		url, err := urllib.GetURL(strings.Trim(v.Loc, " \t\n"), *navRequest.URL)
		if err != nil {
			continue
		}
		request := httplib.GetCrawlerRequest(enums2.GET, url)
		request.Source = enums2.FromSitemap
		_ = callback(request)
		result = append(result, request)
	}

	for _, v := range sitemap.Sitemap {
		url, err := urllib.GetURL(strings.Trim(v.Loc, " \t\n"), *navRequest.URL)
		if err != nil {
			continue
		}
		request := httplib.GetCrawlerRequest(enums2.GET, url)
		request.Source = enums2.FromSitemap
		_ = callback(request)
		result = append(result, request)
	}
	return result, nil
}

// DoFuzzFromDefaultDict 做目录fuzz从默认字典
func (expression *CrawlerExpression) DoFuzzFromDefaultDict(navRequest httplib.RequestCrawler, callback func(i *httplib.RequestCrawler) error) ([]*httplib.RequestCrawler, error) {
	return expression.DoDictRequestFuzz(navRequest, strings.Split(enums2.DefaultFuzzDict, "/"), callback)
}

// DoFuzzFromCustomDict 做目录fuzz从用户自定义字典
func (expression *CrawlerExpression) DoFuzzFromCustomDict(navRequest httplib.RequestCrawler, callback func(i *httplib.RequestCrawler) error, dictPath string) ([]*httplib.RequestCrawler, error) {
	return expression.DoDictRequestFuzz(navRequest, utils.ReadFile(dictPath), callback)
}

// DoDictRequestFuzz 做请求的fuzz目录操作
func (expression *CrawlerExpression) DoDictRequestFuzz(navRequest httplib.RequestCrawler, paths []string, callback func(i *httplib.RequestCrawler) error) ([]*httplib.RequestCrawler, error) {
	expression.FuzzValidateUrlList = mapset.NewSet()
	var result []*httplib.RequestCrawler
	pool, _ := ants.NewPool(20)
	defer pool.Release()
	for _, path := range paths {
		path = strings.TrimPrefix(path, "/")
		path = strings.TrimSuffix(path, "\n")
		task := FuzzSingle{request: navRequest, path: path, fuzzWaitGroup: &expression.FuzzWaitGroup, fuzzValidateUrlList: expression.FuzzValidateUrlList}
		expression.FuzzWaitGroup.Add(1)
		go func() {
			err := pool.Submit(task.DoHttpRequest)
			if err != nil {
				expression.FuzzWaitGroup.Done()
			}
		}()
	}
	expression.FuzzWaitGroup.Wait()
	for _, _url := range expression.FuzzValidateUrlList.ToSlice() {
		url, err := urllib.GetURL(_url.(string))
		if err != nil {
			continue
		}
		req := httplib.GetCrawlerRequest(enums2.GET, url)
		req.Source = enums2.FromFuzz
		_ = callback(req)
		result = append(result, req)
	}
	return result, nil
}

// DoHttpRequest 做一个fuzz请求
func (single *FuzzSingle) DoHttpRequest() {
	defer single.fuzzWaitGroup.Done()
	resp, errs := requests.Get(fmt.Sprintf(`%s://%s/%s`, single.request.URL.Scheme, single.request.URL.Host, single.path), utils.ConvertHeaders(single.request.Headers),
		&requests.RequestOptions{Timeout: 2, AllowRedirect: false, Proxy: single.request.Proxy})
	if errs != nil {
		return
	}
	//fmt.Println(fmt.Sprintf(`%s://%s/%s`, single.request.URL.Scheme, single.request.URL.Host, single.path), resp.StatusCode)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		single.fuzzValidateUrlList.Add(fmt.Sprintf(`%s://%s/%s`, single.request.URL.Scheme, single.request.URL.Host, single.path))
	} else if resp.StatusCode == 301 {
		Locations := resp.Header["Location"]
		if len(Locations) <= 0 {
			return
		}
		redirectUrl, err := urllib.GetURL(Locations[0])
		if err != nil {
			return
		}
		if redirectUrl.Host == single.request.URL.Host {
			single.fuzzValidateUrlList.Add(fmt.Sprintf(`%s://%s/%s`, single.request.URL.Scheme, single.request.URL.Host, single.path))
		}
	}
}
