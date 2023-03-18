package expression

import (
	mapset "github.com/deckarep/golang-set"
	"github.com/sairson/crawlergo/internal/engine/httplib"
	"sync"
)

type CrawlerExpression struct {
	FuzzWaitGroup       sync.WaitGroup
	FuzzValidateUrlList mapset.Set
}

type Sitemap struct {
	URLs    []LocUrl `xml:"url"`
	Sitemap []LocUrl `xml:"sitemap"`
}

type LocUrl struct {
	Loc string `xml:"loc"`
}

type FuzzSingle struct {
	path                string
	fuzzWaitGroup       *sync.WaitGroup
	request             httplib.RequestCrawler
	fuzzValidateUrlList mapset.Set
}
