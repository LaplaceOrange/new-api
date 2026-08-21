package setting

import (
	"fmt"
	"net/url"
	"strings"
)

var RegionRestrictionEnabled = false
var RegionRestrictionCountries = ""
var RegionRestrictionRedirectURL = "https://example.com"

func NormalizeRegionRestrictionCountries(value string) (string, error) {
	parts := strings.Split(value, ",")
	seen := make(map[string]struct{}, len(parts))
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		country := strings.ToUpper(strings.TrimSpace(part))
		if country == "" {
			continue
		}
		if len(country) != 2 || country[0] < 'A' || country[0] > 'Z' || country[1] < 'A' || country[1] > 'Z' {
			return "", fmt.Errorf("invalid country code %q; use two-letter ISO codes separated by commas", part)
		}
		if _, ok := seen[country]; ok {
			continue
		}
		seen[country] = struct{}{}
		result = append(result, country)
	}
	return strings.Join(result, ","), nil
}

func ValidateRegionRestrictionRedirectURL(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("redirect URL cannot be empty")
	}
	u, err := url.ParseRequestURI(trimmed)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("redirect URL must be an absolute http or https URL")
	}
	return nil
}
