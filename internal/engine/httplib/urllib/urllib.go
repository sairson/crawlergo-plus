package urllib

import (
	"errors"
	"fmt"
	"github.com/sairson/crawlergo/pkg/urllib"
	"golang.org/x/net/publicsuffix"
	"net/url"
	"path"
	"regexp"
	"strings"
)

type URL struct {
	url.URL
}

// GetURL 解析url生成完整格式
func GetURL(_url string, parentUrls ...URL) (*URL, error) {
	var u URL
	if _url, err := u.parse(_url, parentUrls...); err != nil {
		return nil, err
	} else {
		if len(parentUrls) == 0 {
			if _u, err := urllib.UrlParse(_url); err != nil {
				return nil, err
			} else {
				u = URL{*_u}
				if u.Path == "" {
					u.Path = "/"
				}
			}
		}
		if len(parentUrls) > 0 {
			pUrl := parentUrls[0]
			_u, err := pUrl.Parse(_url)
			if err != nil {
				return nil, err
			}
			u = URL{*_u}
			if u.Path == "" {
				u.Path = "/"
			}
		}
		fixPath := regexp.MustCompile("^/{2,}")
		if fixPath.MatchString(u.Path) {
			u.Path = fixPath.ReplaceAllString(u.Path, "/")
		}
		return &u, nil
	}
}

// parse 修复不完整的url
func (u *URL) parse(_url string, parentUrls ...URL) (string, error) {
	_url = strings.Trim(_url, " ")
	if len(_url) == 0 {
		return "", errors.New("invalid url,url length is zero")
	}
	// 替换掉多余的#
	if strings.Count(_url, "#") > 1 {
		_url = regexp.MustCompile(`#+`).ReplaceAllString(_url, "#")
	}
	// 没有父链接，直接退出
	if len(parentUrls) == 0 {
		return _url, nil
	}
	if strings.HasPrefix(_url, "http://") || strings.HasPrefix(_url, "https://") {
		return _url, nil
	} else if strings.HasPrefix(_url, "javascript:") {
		return "", errors.New("invalid url, javascript protocol")
	} else if strings.HasPrefix(_url, "mailto:") {
		return "", errors.New("invalid url, mailto protocol")
	}
	return _url, nil
}

// QueryMap 将url的每一个参数转为key value格式
func (u *URL) QueryMap() map[string]interface{} {
	queryMap := map[string]interface{}{}
	for key, value := range u.Query() {
		if len(value) == 1 {
			queryMap[key] = value[0]
		} else {
			queryMap[key] = value
		}
	}
	return queryMap
}

// NoQueryUrl  去掉所有参数的url
func (u *URL) NoQueryUrl() string {
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path)
}

// NoFragmentUrl 返回不带Fragment的URL Fragment #
func (u *URL) NoFragmentUrl() string {
	return strings.Replace(u.String(), u.Fragment, "", -1)
}

// NoSchemeFragmentUrl  返回没有协议格式的url
func (u *URL) NoSchemeFragmentUrl() string {
	return fmt.Sprintf("://%s%s", u.Host, u.Path)
}

// NavigationUrl  返回没有协议格式的url
func (u *URL) NavigationUrl() string {
	return u.NoSchemeFragmentUrl()
}

// RootDomain 返回根域名
func (u *URL) RootDomain() string {
	domain := u.Hostname()
	suffix, icann := publicsuffix.PublicSuffix(strings.ToLower(domain))
	// 如果不是 icann 的域名，返回空字符串
	if !icann {
		return ""
	}
	i := len(domain) - len(suffix) - 1
	// 如果域名错误
	if i <= 0 {
		return ""
	}
	if domain[i] != '.' {
		return ""
	}
	return domain[1+strings.LastIndex(domain[:i], "."):]
}

// FileName  返回文件名
func (u *URL) FileName() string {
	parts := strings.Split(u.Path, `/`)
	lastPart := parts[len(parts)-1]
	if strings.Contains(lastPart, ".") {
		return lastPart
	} else {
		return ""
	}
}

// FileExt 返回文件拓展名
func (u *URL) FileExt() string {
	parts := path.Ext(u.Path)
	// 第一个字符会带有 "."
	if len(parts) > 0 {
		return strings.ToLower(parts[1:])
	}
	return parts
}

// ParentPath 返回上一级路径path，如果没有的话，就返回空字符串
func (u *URL) ParentPath() string {
	if u.Path == "/" {
		return ""
	} else if strings.HasSuffix(u.Path, "/") {
		if strings.Count(u.Path, "/") == 2 {
			return "/"
		}
		parts := strings.Split(u.Path, "/")
		parts = parts[:len(parts)-2]
		return strings.Join(parts, "/")
	} else {
		if strings.Count(u.Path, "/") == 1 {
			return "/"
		}
		parts := strings.Split(u.Path, "/")
		parts = parts[:len(parts)-1]
		return strings.Join(parts, "/")
	}
}
