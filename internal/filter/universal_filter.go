package filter

import (
	mapset "github.com/deckarep/golang-set"
	"github.com/sairson/crawlergo/internal/engine/enums"
	"github.com/sairson/crawlergo/internal/engine/httplib"
	"github.com/sairson/crawlergo/internal/engine/httplib/urllib"
	"github.com/sairson/crawlergo/internal/option"
	"github.com/sairson/crawlergo/pkg/utils"
	"go/types"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// SmartFilter 智能过滤器
type SmartFilter struct {
	StrictMode                 bool
	SimpleFilter               SimpleFilter
	filterLocationSet          mapset.Set // 非逻辑型参数的位置记录 全局统一标记过滤
	filterParamKeyRepeatCount  sync.Map
	filterParamKeySingleValues sync.Map // 所有参数名重复数量统计
	filterPathParamKeySymbol   sync.Map // 某个path下的某个参数的值出现标记次数统计
	filterParamKeyAllValues    sync.Map
	filterPathParamEmptyValues sync.Map
	filterParentPathValues     sync.Map
	uniqueMarkedIds            mapset.Set // 标记后的唯一ID，用于去重
}

type SimpleFilter struct {
	UniqueSet mapset.Set
	HostLimit string
}

var (
	staticSuffixSet = option.StaticSuffixSet.Clone()
)

func (s *SmartFilter) Init() {
	s.filterLocationSet = mapset.NewSet()
	s.filterParamKeyRepeatCount = sync.Map{}
	s.filterParamKeySingleValues = sync.Map{}
	s.filterPathParamKeySymbol = sync.Map{}
	s.filterParamKeyAllValues = sync.Map{}
	s.filterPathParamEmptyValues = sync.Map{}
	s.filterParentPathValues = sync.Map{}
	s.uniqueMarkedIds = mapset.NewSet()
}

// DoFilter 做普通模式的过滤操作
func (s *SimpleFilter) DoFilter(req *httplib.RequestCrawler) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	// 首先判断是否需要过滤域名
	if s.HostLimit != "" && s.DomainFilter(req) {
		return true
	}
	// 去重过滤
	if s.UniqueFilter(req) {
		return true
	}
	// 过滤静态资源
	if s.StaticFilter(req) {
		return true
	}
	return false
}

// UniqueFilter 请求去重
func (s *SimpleFilter) UniqueFilter(req *httplib.RequestCrawler) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	if s.UniqueSet.Contains(req.UniqueId()) {
		return true
	} else {
		s.UniqueSet.Add(req.UniqueId())
		return false
	}
}

// DomainFilter 域名过滤
func (s *SimpleFilter) DomainFilter(req *httplib.RequestCrawler) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	if req.URL.Host == s.HostLimit || req.URL.Hostname() == s.HostLimit {
		return false
	}
	if strings.HasSuffix(s.HostLimit, ":80") && req.URL.Port() == "" && req.URL.Scheme == "http" {
		if req.URL.Hostname()+":80" == s.HostLimit {
			return false
		}
	}
	if strings.HasSuffix(s.HostLimit, ":443") && req.URL.Port() == "" && req.URL.Scheme == "https" {
		if req.URL.Hostname()+":443" == s.HostLimit {
			return false
		}
	}
	return true
}

// StaticFilter 静态资源过滤
func (s *SimpleFilter) StaticFilter(req *httplib.RequestCrawler) bool {
	if s.UniqueSet == nil {
		s.UniqueSet = mapset.NewSet()
	}
	// 首先将slice转换成map
	if req.URL.FileExt() == "" {
		return false
	}
	if staticSuffixSet.Contains(req.URL.FileExt()) {
		return true
	}
	return false
}

func init() {
	for _, suffix := range []string{"js", "css", "json"} {
		staticSuffixSet.Add(suffix)
	}
}

func (s *SmartFilter) DoFilter(req *httplib.RequestCrawler) bool {
	if s.SimpleFilter.DoFilter(req) {
		return true
	}
	req.Filter.FragmentID = s.CalcFragmentID(req.URL.Fragment)

	// 标记
	if req.Method == enums.GET || req.Method == enums.DELETE || req.Method == enums.HEAD || req.Method == enums.OPTIONS {
		s.GetMark(*req)
		s.repeatCountStatistic(req)
	} else if req.Method == enums.POST || req.Method == enums.PUT {
		s.postMark(req)
	} else {
		// 不做处理
	}
	// 对标记后的请求进行去重
	uniqueId := req.Filter.UniqueId
	if s.uniqueMarkedIds.Contains(uniqueId) {
		return true
	}

	// 全局数值型参数标记
	s.globalFilterLocationMark(req)

	// 接下来对标记的GET请求进行去重
	if req.Method == enums.GET || req.Method == enums.DELETE || req.Method == enums.HEAD || req.Method == enums.OPTIONS {
		// 对超过阈值的GET请求进行标记
		s.overCountMark(req)

		// 重新计算 QueryMapId
		req.Filter.QueryMapId = s.getParamMapID(req.Filter.MarkedQueryMap)
		// 重新计算 PathId
		req.Filter.PathId = s.getPathID(req.Filter.MarkedPath)
	} else {
		// 重新计算 PostDataId
		req.Filter.PostDataId = s.getParamMapID(req.Filter.MarkedPostDataMap)
	}

	// 重新计算请求唯一ID
	req.Filter.UniqueId = s.getMarkedUniqueID(req)

	// 新的ID再次去重
	newUniqueId := req.Filter.UniqueId
	if s.uniqueMarkedIds.Contains(newUniqueId) {
		return true
	}

	// 添加到结果集中
	s.uniqueMarkedIds.Add(newUniqueId)
	return false
}

func (s *SmartFilter) CalcFragmentID(fragment string) string {
	if fragment == "" || !strings.HasPrefix(fragment, "/") {
		return ""
	}
	fakeUrl, err := urllib.GetURL(fragment)
	if err != nil {
		return ""
	}
	fakeRequest := httplib.GetCrawlerRequest(enums.GET, fakeUrl)
	s.GetMark(*fakeRequest)
	return fakeRequest.Filter.UniqueId
}

// GetMark 为请求打标记
func (s *SmartFilter) GetMark(req httplib.RequestCrawler) {
	// 解码前的预先替换
	todoUrl := *(req.URL)
	todoUrl.RawQuery = s.preQueryMark(todoUrl.RawQuery)
	// 依次打标记
	queryMap := todoUrl.QueryMap()
	queryMap = s.markParamName(queryMap)
	queryMap = s.markParamValue(queryMap, &req)
	markedPath := s.MarkPath(todoUrl.Path)
	// 计算唯一的ID
	var queryKeyID string
	var queryMapID string
	if len(queryMap) != 0 {
		queryKeyID = s.getKeysID(queryMap)
		queryMapID = s.getParamMapID(queryMap)
	} else {
		queryKeyID = ""
		queryMapID = ""
	}
	pathID := s.getPathID(markedPath)

	req.Filter.MarkedQueryMap = queryMap
	req.Filter.QueryKeysId = queryKeyID
	req.Filter.QueryMapId = queryMapID
	req.Filter.MarkedPath = markedPath
	req.Filter.PathId = pathID

	// 最后计算标记后的唯一请求ID
	req.Filter.UniqueId = s.getMarkedUniqueID(&req)
}

// preQueryMark Query的Map对象会自动解码，所以对RawQuery进行预先的标记
func (s *SmartFilter) preQueryMark(rawQuery string) string {
	if enums.ChineseRegex.MatchString(rawQuery) {
		return enums.ChineseRegex.ReplaceAllString(rawQuery, enums.ChineseMark)
	} else if enums.UrlencodeRegex.MatchString(rawQuery) {
		return enums.UrlencodeRegex.ReplaceAllString(rawQuery, enums.UrlEncodeMark)
	} else if enums.UnicodeRegex.MatchString(rawQuery) {
		return enums.UnicodeRegex.ReplaceAllString(rawQuery, enums.UnicodeMark)
	}
	return rawQuery
}

func (s *SmartFilter) markParamName(paramMap map[string]interface{}) map[string]interface{} {
	markedParamMap := map[string]interface{}{}
	for key, value := range paramMap {
		// 纯字母不处理
		if enums.OnlyAlphaRegex.MatchString(key) {
			markedParamMap[key] = value
			// 参数名过长
		} else if len(key) >= 32 {
			markedParamMap[enums.TooLongMark] = value
			// 替换掉数字
		} else {
			key = enums.ReplaceNumRegex.ReplaceAllString(key, enums.NumberMark)
			markedParamMap[key] = value
		}
	}
	return markedParamMap
}

func (s *SmartFilter) markParamValue(paramMap map[string]interface{}, req *httplib.RequestCrawler) map[string]interface{} {
	markedParamMap := map[string]interface{}{}
	for key, value := range paramMap {
		switch value.(type) {
		case bool:
			markedParamMap[key] = enums.BoolMark
			continue
		case types.Slice:
			markedParamMap[key] = enums.ListMark
			continue
		case float64:
			markedParamMap[key] = enums.NumberMark
			continue
		}
		// 只处理string类型
		valueStr, ok := value.(string)
		if !ok {
			continue
		}
		// Custom 为特定字符，说明此参数位置为数值型，非逻辑型，记录下此参数，全局过滤
		if strings.Contains(valueStr, "Custom") {
			name := req.URL.Hostname() + req.URL.Path + req.Method + key
			s.filterLocationSet.Add(name)
			markedParamMap[key] = enums.CustomValueMark
			// 全大写字母
		} else if enums.OnlyAlphaUpperRegex.MatchString(valueStr) {
			markedParamMap[key] = enums.UpperMark
			// 参数值长度大于等于16
		} else if len(valueStr) >= 16 {
			markedParamMap[key] = enums.TooLongMark
			// 均为数字和一些符号组成
		} else if enums.OnlyNumberRegex.MatchString(valueStr) || enums.OnlyNumberRegex.MatchString(enums.NumSymbolRegex.ReplaceAllString(valueStr, "")) {
			markedParamMap[key] = enums.NumberMark
			// 存在中文
		} else if enums.ChineseRegex.MatchString(valueStr) {
			markedParamMap[key] = enums.ChineseMark
			// urlencode
		} else if enums.UrlencodeRegex.MatchString(valueStr) {
			markedParamMap[key] = enums.UrlEncodeMark
			// unicode
		} else if enums.UnicodeRegex.MatchString(valueStr) {
			markedParamMap[key] = enums.UnicodeMark
			// 时间
		} else if enums.OnlyNumberRegex.MatchString(enums.TimeSymbolRegex.ReplaceAllString(valueStr, "")) {
			markedParamMap[key] = enums.TimeMark
			// 字母加数字
		} else if enums.OnlyAlphaNumRegex.MatchString(valueStr) && enums.NumberRegex.MatchString(valueStr) {
			markedParamMap[key] = enums.MixAlphaNumMark
			// 含有一些特殊符号
		} else if s.hasSpecialSymbol(valueStr) {
			markedParamMap[key] = enums.MixSymbolMark
			// 数字出现的次数超过3，视为数值型参数
		} else if b := enums.OneNumberRegex.ReplaceAllString(valueStr, "0"); strings.Count(b, "0") >= 3 {
			markedParamMap[key] = enums.MixNumMark
			// 严格模式
		} else if s.StrictMode {
			// 无小写字母
			if !enums.AlphaLowerRegex.MatchString(valueStr) {
				markedParamMap[key] = enums.NoLowerAlphaMark
				// 常见的值一般为 大写字母、小写字母、数字、下划线的任意组合，组合类型超过三种则视为伪静态
			} else {
				count := 0
				if enums.AlphaLowerRegex.MatchString(valueStr) {
					count += 1
				}
				if enums.AlphaUpperRegex.MatchString(valueStr) {
					count += 1
				}
				if enums.NumberRegex.MatchString(valueStr) {
					count += 1
				}
				if strings.Contains(valueStr, "_") || strings.Contains(valueStr, "-") {
					count += 1
				}
				if count >= 3 {
					markedParamMap[key] = enums.MixStringMark
				}
			}
		} else {
			markedParamMap[key] = value
		}
	}
	return markedParamMap
}

func (s *SmartFilter) hasSpecialSymbol(str string) bool {
	symbolList := []string{"{", "}", " ", "|", "#", "@", "$", "*", ",", "<", ">", "/", "?", "\\", "+", "="}
	for _, sym := range symbolList {
		if strings.Contains(str, sym) {
			return true
		}
	}
	return false
}

func (s *SmartFilter) MarkPath(path string) string {
	pathParts := strings.Split(path, "/")
	for index, part := range pathParts {
		if len(part) >= 32 {
			pathParts[index] = enums.TooLongMark
		} else if enums.OnlyNumberRegex.MatchString(enums.NumSymbolRegex.ReplaceAllString(part, "")) {
			pathParts[index] = enums.NumberMark
		} else if strings.HasSuffix(part, ".html") || strings.HasSuffix(part, ".htm") || strings.HasSuffix(part, ".shtml") {
			part = enums.HtmlReplaceRegex.ReplaceAllString(part, "")
			// 大写、小写、数字混合
			if enums.NumberRegex.MatchString(part) && enums.AlphaUpperRegex.MatchString(part) && enums.AlphaLowerRegex.MatchString(part) {
				pathParts[index] = enums.MixAlphaNumMark
				// 纯数字
			} else if b := enums.NumSymbolRegex.ReplaceAllString(part, ""); enums.OnlyNumberRegex.MatchString(b) {
				pathParts[index] = enums.NumberMark
			}
			// 含有特殊符号
		} else if s.hasSpecialSymbol(part) {
			pathParts[index] = enums.MixSymbolMark
		} else if enums.ChineseRegex.MatchString(part) {
			pathParts[index] = enums.ChineseMark
		} else if enums.UnicodeRegex.MatchString(part) {
			pathParts[index] = enums.UnicodeMark
		} else if enums.OnlyAlphaUpperRegex.MatchString(part) {
			pathParts[index] = enums.UpperMark
			// 均为数字和一些符号组成
		} else if b := enums.NumSymbolRegex.ReplaceAllString(part, ""); enums.OnlyNumberRegex.MatchString(b) {
			pathParts[index] = enums.NumberMark
			// 数字出现的次数超过3，视为伪静态path
		} else if b := enums.OneNumberRegex.ReplaceAllString(part, "0"); strings.Count(b, "0") > 3 {
			pathParts[index] = enums.MixNumMark
		}
	}
	newPath := strings.Join(pathParts, "/")
	return newPath
}

func (s *SmartFilter) getKeysID(dataMap map[string]interface{}) string {
	var keys []string
	var idStr string
	for key := range dataMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		idStr += key
	}
	return utils.CalcMD5Hash(idStr)
}

func (s *SmartFilter) getParamMapID(dataMap map[string]interface{}) string {
	var keys []string
	var idStr string
	var markReplaceRegex = regexp.MustCompile(`{{.+}}`)
	for key := range dataMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := dataMap[key]
		idStr += key
		if value, ok := value.(string); ok {
			idStr += markReplaceRegex.ReplaceAllString(value, "{{mark}}")
		}
	}
	return utils.CalcMD5Hash(idStr)
}

func (s *SmartFilter) getPathID(path string) string {
	return utils.CalcMD5Hash(path)
}

func (s *SmartFilter) getMarkedUniqueID(req *httplib.RequestCrawler) string {
	var paramId string
	if req.Method == enums.GET || req.Method == enums.DELETE || req.Method == enums.HEAD || req.Method == enums.OPTIONS {
		paramId = req.Filter.QueryMapId
	} else {
		paramId = req.Filter.PostDataId
	}

	uniqueStr := req.Method + paramId + req.Filter.PathId + req.URL.Host + req.Filter.FragmentID
	if req.Redirection {
		uniqueStr += "Redirection"
	}
	if req.URL.Path == "/" && req.URL.RawQuery == "" && req.URL.Scheme == "https" {
		uniqueStr += "https"
	}

	return utils.CalcMD5Hash(uniqueStr)
}

func (s *SmartFilter) repeatCountStatistic(req *httplib.RequestCrawler) {
	queryKeyId := req.Filter.QueryKeysId
	pathId := req.Filter.PathId
	if queryKeyId != "" {
		// 所有参数名重复数量统计
		if v, ok := s.filterParamKeyRepeatCount.Load(queryKeyId); ok {
			s.filterParamKeyRepeatCount.Store(queryKeyId, v.(int)+1)
		} else {
			s.filterParamKeyRepeatCount.Store(queryKeyId, 1)
		}

		for key, value := range req.Filter.MarkedQueryMap {
			// 某个URL的所有参数名重复数量统计
			paramQueryKey := queryKeyId + key

			if set, ok := s.filterParamKeySingleValues.Load(paramQueryKey); ok {
				set := set.(mapset.Set)
				set.Add(value)
			} else {
				s.filterParamKeySingleValues.Store(paramQueryKey, mapset.NewSet(value))
			}

			//本轮所有URL中某个参数重复数量统计
			if _, ok := s.filterParamKeyAllValues.Load(key); !ok {
				s.filterParamKeyAllValues.Store(key, mapset.NewSet(value))
			} else {
				if v, ok := s.filterParamKeyAllValues.Load(key); ok {
					set := v.(mapset.Set)
					if !set.Contains(value) {
						set.Add(value)
					}
				}
			}

			// 如果参数值为空，统计该PATH下的空值参数名个数
			if value == "" {
				if _, ok := s.filterPathParamEmptyValues.Load(pathId); !ok {
					s.filterPathParamEmptyValues.Store(pathId, mapset.NewSet(key))
				} else {
					if v, ok := s.filterPathParamEmptyValues.Load(pathId); ok {
						set := v.(mapset.Set)
						if !set.Contains(key) {
							set.Add(key)
						}
					}
				}
			}

			pathIdKey := pathId + key
			// 某path下的参数值去重标记出现次数统计
			if v, ok := s.filterPathParamKeySymbol.Load(pathIdKey); ok {
				if enums.MarkedStringRegex.MatchString(value.(string)) {
					s.filterPathParamKeySymbol.Store(pathIdKey, v.(int)+1)
				}
			} else {
				s.filterPathParamKeySymbol.Store(pathIdKey, 1)
			}

		}
	}

	// 相对于上一级目录，本级path目录的数量统计，存在文件后缀的情况下，放行常见脚本后缀
	if req.URL.ParentPath() == "" || s.inCommonScriptSuffix(req.URL.FileExt()) {
		return
	}

	parentPathId := utils.CalcMD5Hash(req.URL.ParentPath())
	currentPath := strings.Replace(req.Filter.MarkedPath, req.URL.ParentPath(), "", -1)
	if _, ok := s.filterParentPathValues.Load(parentPathId); !ok {
		s.filterParentPathValues.Store(parentPathId, mapset.NewSet(currentPath))
	} else {
		if v, ok := s.filterParentPathValues.Load(parentPathId); ok {
			set := v.(mapset.Set)
			if !set.Contains(currentPath) {
				set.Add(currentPath)
			}
		}
	}
}

func (s *SmartFilter) inCommonScriptSuffix(suffix string) bool {
	return option.ScriptSuffixSet.Contains(suffix)
}

func (s *SmartFilter) postMark(req *httplib.RequestCrawler) {
	postDataMap := req.CrawlerPostData() // 爬虫需要的post data类型

	postDataMap = s.markParamName(postDataMap)
	postDataMap = s.markParamValue(postDataMap, req)
	markedPath := s.MarkPath(req.URL.Path)

	// 计算唯一的ID
	var postDataMapID string
	if len(postDataMap) != 0 {
		postDataMapID = s.getParamMapID(postDataMap)
	} else {
		postDataMapID = ""
	}
	pathID := s.getPathID(markedPath)

	req.Filter.MarkedPostDataMap = postDataMap
	req.Filter.PostDataId = postDataMapID
	req.Filter.MarkedPath = markedPath
	req.Filter.PathId = pathID

	// 最后计算标记后的唯一请求ID
	req.Filter.UniqueId = s.getMarkedUniqueID(req)
}

func (s *SmartFilter) globalFilterLocationMark(req *httplib.RequestCrawler) {
	name := req.URL.Hostname() + req.URL.Path + req.Method
	if req.Method == enums.GET || req.Method == enums.DELETE || req.Method == enums.HEAD || req.Method == enums.OPTIONS {
		for key := range req.Filter.MarkedQueryMap {
			name += key
			if s.filterLocationSet.Contains(name) {
				req.Filter.MarkedQueryMap[key] = enums.CustomValueMark
			}
		}
	} else if req.Method == enums.POST || req.Method == enums.PUT {
		for key := range req.Filter.MarkedPostDataMap {
			name += key
			if s.filterLocationSet.Contains(name) {
				req.Filter.MarkedPostDataMap[key] = enums.CustomValueMark
			}
		}
	}
}

func (s *SmartFilter) overCountMark(req *httplib.RequestCrawler) {
	queryKeyId := req.Filter.QueryKeysId
	pathId := req.Filter.PathId
	// 参数不为空，
	if req.Filter.QueryKeysId != "" {
		// 某个URL的所有参数名重复数量超过阈值 且该参数有超过三个不同的值 则打标记
		if v, ok := s.filterParamKeyRepeatCount.Load(queryKeyId); ok && v.(int) > enums.MaxParamKeySingleCount {
			for key := range req.Filter.MarkedQueryMap {
				paramQueryKey := queryKeyId + key
				if set, ok := s.filterParamKeySingleValues.Load(paramQueryKey); ok {
					set := set.(mapset.Set)
					if set.Cardinality() > 3 {
						req.Filter.MarkedQueryMap[key] = enums.FixParamRepeatMark
					}
				}
			}
		}

		for key := range req.Filter.MarkedQueryMap {
			// 所有URL中，某个参数不同的值出现次数超过阈值，打标记去重
			if paramKeySet, ok := s.filterParamKeyAllValues.Load(key); ok {
				paramKeySet := paramKeySet.(mapset.Set)
				if paramKeySet.Cardinality() > enums.MaxParamKeyAllCount {
					req.Filter.MarkedQueryMap[key] = enums.FixParamRepeatMark
				}
			}

			pathIdKey := pathId + key
			// 某个PATH的GET参数值去重标记出现次数超过阈值，则对该PATH的该参数进行全局标记
			if v, ok := s.filterPathParamKeySymbol.Load(pathIdKey); ok && v.(int) > enums.MaxPathParamKeySymbolCount {
				req.Filter.MarkedQueryMap[key] = enums.FixParamRepeatMark
			}
		}

		// 处理某个path下空参数值的参数个数超过阈值 如伪静态： http://bang.360.cn/?chu_xiu
		if v, ok := s.filterPathParamEmptyValues.Load(pathId); ok {
			set := v.(mapset.Set)
			if set.Cardinality() > enums.MaxPathParamEmptyCount {
				newMarkerQueryMap := map[string]interface{}{}
				for key, value := range req.Filter.MarkedQueryMap {
					if value == "" {
						newMarkerQueryMap[enums.FixParamRepeatMark] = ""
					} else {
						newMarkerQueryMap[key] = value
					}
				}
				req.Filter.MarkedQueryMap = newMarkerQueryMap
			}
		}
	}

	// 处理本级path的伪静态
	if req.URL.ParentPath() == "" || s.inCommonScriptSuffix(req.URL.FileExt()) {
		return
	}
	parentPathId := utils.CalcMD5Hash(req.URL.ParentPath())
	if set, ok := s.filterParentPathValues.Load(parentPathId); ok {
		set := set.(mapset.Set)
		if set.Cardinality() > enums.MaxParentPathCount {
			if strings.HasSuffix(req.URL.ParentPath(), "/") {
				req.Filter.MarkedPath = req.URL.ParentPath() + enums.FixPathMark
			} else {
				req.Filter.MarkedPath = req.URL.ParentPath() + "/" + enums.FixPathMark
			}
		}
	}
}
