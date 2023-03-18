package httplib

import (
	"github.com/axgle/mahonia"
	"github.com/saintfish/chardet"
	"net/http"
)

type ResponseCrawler struct {
	http.Response
	Body     []byte // 原始文本对于response
	Encoding string // 文本编码类型
}

// ToText 返回受编码的文本
func (resp *ResponseCrawler) ToText() string {
	if resp.Encoding != "" {
		enc := mahonia.NewEncoder(resp.Encoding)
		return enc.ConvertString(string(resp.Body))
	} else {
		enc := mahonia.NewEncoder("UTF-8")
		return enc.ConvertString(string(resp.Body))
	}
}

func DetectEncoding(content []byte) (string, error) {
	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(content)
	return result.Charset, err
}
