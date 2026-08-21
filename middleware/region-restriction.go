package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

const (
	regionLookupURL = "http://ip.taobao.com/service/getIpInfo.php?ip=%s"
	regionCacheTTL  = 10 * time.Minute
)

type taobaoIPInfoResponse struct {
	Code int `json:"code"`
	Data struct {
		CountryID string `json:"country_id"`
	} `json:"data"`
}

type regionCacheEntry struct {
	country string
	expires time.Time
}

var regionCache = struct {
	sync.RWMutex
	entries map[string]regionCacheEntry
}{entries: make(map[string]regionCacheEntry)}

// RegionRestrictionGuard looks up the client IP through Taobao's IP service
// before normal routing when the request country is listed in the settings.
func RegionRestrictionGuard() gin.HandlerFunc {
	client := &http.Client{Timeout: 2 * time.Second}
	return func(c *gin.Context) {
		if !setting.RegionRestrictionEnabled {
			c.Next()
			return
		}
		country := lookupRegionCountry(client, c.ClientIP())
		if country == "" || !restrictedCountry(country) {
			c.Next()
			return
		}
		url := strings.TrimSpace(setting.RegionRestrictionRedirectURL)
		if url == "" {
			c.String(http.StatusForbidden, "Access restricted")
			c.Abort()
			return
		}
		c.Redirect(http.StatusTemporaryRedirect, url)
		c.Abort()
	}
}

func lookupRegionCountry(client *http.Client, ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	now := time.Now()
	regionCache.RLock()
	entry, ok := regionCache.entries[ip]
	regionCache.RUnlock()
	if ok && now.Before(entry.expires) {
		return entry.country
	}
	response, err := client.Get(fmt.Sprintf(regionLookupURL, url.QueryEscape(ip)))
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ""
	}
	var payload taobaoIPInfoResponse
	if err := common.DecodeJson(response.Body, &payload); err != nil || payload.Code != 0 {
		return ""
	}
	country := strings.ToUpper(strings.TrimSpace(payload.Data.CountryID))
	regionCache.Lock()
	regionCache.entries[ip] = regionCacheEntry{country: country, expires: now.Add(regionCacheTTL)}
	regionCache.Unlock()
	return country
}

func restrictedCountry(country string) bool {
	for _, value := range strings.Split(setting.RegionRestrictionCountries, ",") {
		if strings.EqualFold(strings.TrimSpace(value), country) {
			return true
		}
	}
	return false
}
