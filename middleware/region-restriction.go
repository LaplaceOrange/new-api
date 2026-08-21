package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/pkg/ip2region"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

const regionCacheTTL = 10 * time.Minute

type regionCacheEntry struct {
	country string
	expires time.Time
}

var regionCache = struct {
	sync.RWMutex
	entries map[string]regionCacheEntry
}{entries: make(map[string]regionCacheEntry)}

// RegionRestrictionGuard looks up the client IP through the embedded ip2region database
// before normal routing when the request country is listed in the settings.
func RegionRestrictionGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !setting.RegionRestrictionEnabled {
			c.Next()
			return
		}
		country := lookupRegionCountry(c.ClientIP())
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

func lookupRegionCountry(ip string) string {
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
	country, err := ip2region.Country(ip)
	if err != nil {
		return ""
	}
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
