package expression

import (
	"fmt"
	"github.com/sairson/crawlergo/internal/engine/httplib"
	"github.com/sairson/crawlergo/internal/engine/httplib/urllib"
	"testing"
)

func TestCrawlerExpression_Robots(t *testing.T) {
	var expression CrawlerExpression
	url, err := urllib.GetURL("https://security-crawl-maze.app/")
	if err != nil {
		return
	}
	fmt.Println(expression.Robots(httplib.RequestCrawler{
		URL:         url,
		Method:      "GET",
		Headers:     nil,
		PostData:    "",
		Filter:      httplib.Filter{},
		Source:      "",
		Redirection: false,
		Proxy:       "",
	}, func(i *httplib.RequestCrawler) error {
		fmt.Println(i)
		return nil
	}))
}

func TestCrawlerExpression_Sitemap(t *testing.T) {
	var expression CrawlerExpression
	url, err := urllib.GetURL("https://security-crawl-maze.app/")
	if err != nil {
		return
	}
	fmt.Println(expression.Sitemap(httplib.RequestCrawler{
		URL:         url,
		Method:      "GET",
		Headers:     nil,
		PostData:    "",
		Filter:      httplib.Filter{},
		Source:      "",
		Redirection: false,
		Proxy:       "",
	}, func(i *httplib.RequestCrawler) error {
		fmt.Println(i)
		return nil
	}))
}

func TestCrawlerExpression_DoFuzzFromDefaultDict(t *testing.T) {
	var expression CrawlerExpression
	url, err := urllib.GetURL("https://security-crawl-maze.app/")
	if err != nil {
		return
	}
	fmt.Println(expression.DoFuzzFromDefaultDict(httplib.RequestCrawler{
		URL:         url,
		Method:      "GET",
		Headers:     nil,
		PostData:    "",
		Filter:      httplib.Filter{},
		Source:      "",
		Redirection: false,
		Proxy:       "",
	}, func(i *httplib.RequestCrawler) error {
		fmt.Println(i)
		return nil
	}))
}
