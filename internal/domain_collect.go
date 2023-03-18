package internal

import (
	mapset "github.com/deckarep/golang-set"
	"github.com/sairson/crawlergo/internal/engine/httplib"
	"strings"
)

type DomainCollect struct {
}

func (domain *DomainCollect) AllDomainCollect(reqList []*httplib.RequestCrawler) []string {
	uniqueSet := mapset.NewSet()
	var allDomainList []string
	for _, req := range reqList {
		domain := req.URL.Hostname()
		if uniqueSet.Contains(domain) {
			continue
		}
		uniqueSet.Add(domain)
		allDomainList = append(allDomainList, req.URL.Hostname())
	}
	return allDomainList
}

func (domain *DomainCollect) SubDomainCollect(reqList []*httplib.RequestCrawler, HostLimit string) []string {
	var subDomainList []string
	uniqueSet := mapset.NewSet()
	for _, req := range reqList {
		domain := req.URL.Hostname()
		if uniqueSet.Contains(domain) {
			continue
		}
		uniqueSet.Add(domain)
		if strings.HasSuffix(domain, "."+HostLimit) {
			subDomainList = append(subDomainList, domain)
		}
	}
	return subDomainList
}
