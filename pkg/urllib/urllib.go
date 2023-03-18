package urllib

import (
	"fmt"
	"net/url"
	"strings"
)

// UrlParse 调用url.Parse，增加了对%的处理
func UrlParse(sourceUrl string) (*url.URL, error) {
	u, err := url.Parse(sourceUrl)
	if err != nil {
		u, err = url.Parse(EscapePercentSign(sourceUrl))
	}
	if err != nil {
		return nil, fmt.Errorf("url parse has an error %s", err.Error())
	}
	return u, nil
}

// EscapePercentSign  把url中的%替换为%25
func EscapePercentSign(raw string) string {
	return strings.ReplaceAll(raw, "%", "%25")
}
