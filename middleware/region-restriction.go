package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

// RegionRestrictionGuard redirects to the configured external page before
// normal routing when the request country is listed in the restriction settings.
// Country detection is delegated to the edge proxy (Cloudflare and common
// GeoIP proxy headers are supported); direct deployments can set the header
// configured by REGION_COUNTRY_HEADER.
func RegionRestrictionGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !setting.RegionRestrictionEnabled {
			c.Next()
			return
		}
		country := requestCountry(c)
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

func requestCountry(c *gin.Context) string {
	headerName := strings.TrimSpace(os.Getenv("REGION_COUNTRY_HEADER"))
	if headerName != "" {
		if value := strings.TrimSpace(c.GetHeader(headerName)); value != "" {
			return strings.ToUpper(strings.Split(value, ",")[0])
		}
	}
	for _, header := range []string{"CF-IPCountry", "X-Country-Code", "X-GeoIP-Country"} {
		if value := strings.TrimSpace(c.GetHeader(header)); value != "" {
			return strings.ToUpper(strings.Split(value, ",")[0])
		}
	}
	return ""
}

func restrictedCountry(country string) bool {
	for _, value := range strings.Split(setting.RegionRestrictionCountries, ",") {
		if strings.EqualFold(strings.TrimSpace(value), country) {
			return true
		}
	}
	return false
}
