package ip2region

import (
	"embed"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/pkg/ip2region/xdb"
)

//go:embed data/ip2region_v4.xdb data/ip2region_v6.xdb
var database embed.FS

var service struct {
	sync.Once
	v4, v6 *xdb.Searcher
	err    error
}

func initService() {
	service.v4, service.err = newSearcher("data/ip2region_v4.xdb", xdb.IPv4)
	if service.err != nil {
		return
	}
	service.v6, service.err = newSearcher("data/ip2region_v6.xdb", xdb.IPv6)
}

func newSearcher(path string, version *xdb.Version) (*xdb.Searcher, error) {
	content, err := xdb.LoadContentFromFS(database, path)
	if err != nil {
		return nil, err
	}
	return xdb.NewWithBuffer(version, content)
}

func Country(ip string) (string, error) {
	service.Do(initService)
	if service.err != nil {
		return "", service.err
	}
	parsed, err := xdb.ParseIP(ip)
	if err != nil {
		return "", err
	}
	searcher := service.v4
	if len(parsed) == 16 {
		searcher = service.v6
	}
	region, err := searcher.Search(parsed)
	if err != nil {
		return "", err
	}
	return countryCode(strings.TrimSpace(strings.Split(region, "|")[0])), nil
}

func countryCode(value string) string {
	if len(value) == 2 {
		return strings.ToUpper(value)
	}
	return map[string]string{"中国": "CN", "美国": "US", "英国": "GB", "日本": "JP", "韩国": "KR", "加拿大": "CA", "澳大利亚": "AU", "德国": "DE", "法国": "FR", "俄罗斯": "RU", "新加坡": "SG", "印度": "IN"}[value]
}
