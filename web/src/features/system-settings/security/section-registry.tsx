/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { RateLimitSection } from '../request-limits/rate-limit-section'
import { SensitiveWordsSection } from '../request-limits/sensitive-words-section'
import { SSRFSection } from '../request-limits/ssrf-section'
import { TokenLimitSection } from '../request-limits/token-limit-section'
import {
  SECURITY_SECTIONS,
  type SecuritySectionId,
} from '../section-navigation'
import type { SecuritySettings } from '../types'
import { createSectionRegistry } from '../utils/section-registry'
import { RegionRestrictionSection } from './region-restriction-section'

const SECURITY_SECTION_BUILDERS = {
  'rate-limit': (settings: SecuritySettings) => (
    <RateLimitSection
      defaultValues={{
        ModelRequestRateLimitEnabled: settings.ModelRequestRateLimitEnabled,
        ModelRequestRateLimitCount: settings.ModelRequestRateLimitCount,
        ModelRequestRateLimitSuccessCount:
          settings.ModelRequestRateLimitSuccessCount,
        ModelRequestRateLimitDurationMinutes:
          settings.ModelRequestRateLimitDurationMinutes,
        ModelRequestRateLimitGroup: settings.ModelRequestRateLimitGroup,
      }}
    />
  ),
  'sensitive-words': (settings: SecuritySettings) => (
    <SensitiveWordsSection
      defaultValues={{
        CheckSensitiveEnabled: settings.CheckSensitiveEnabled,
        CheckSensitiveOnPromptEnabled: settings.CheckSensitiveOnPromptEnabled,
        SensitiveWordAutoBanEnabled: settings.SensitiveWordAutoBanEnabled,
        SensitiveWordUserBanThreshold: settings.SensitiveWordUserBanThreshold,
        SensitiveWordIPUserBanThreshold:
          settings.SensitiveWordIPUserBanThreshold,
        SensitiveWords: settings.SensitiveWords,
      }}
    />
  ),
  ssrf: (settings: SecuritySettings) => (
    <SSRFSection
      defaultValues={{
        'fetch_setting.enable_ssrf_protection':
          settings['fetch_setting.enable_ssrf_protection'],
        'fetch_setting.allow_private_ip':
          settings['fetch_setting.allow_private_ip'],
        'fetch_setting.domain_filter_mode':
          settings['fetch_setting.domain_filter_mode'],
        'fetch_setting.ip_filter_mode':
          settings['fetch_setting.ip_filter_mode'],
        'fetch_setting.domain_list': settings['fetch_setting.domain_list'],
        'fetch_setting.ip_list': settings['fetch_setting.ip_list'],
        'fetch_setting.allowed_ports': settings['fetch_setting.allowed_ports'],
        'fetch_setting.apply_ip_filter_for_domain':
          settings['fetch_setting.apply_ip_filter_for_domain'],
      }}
    />
  ),
  'region-restriction': (settings: SecuritySettings) => (
    <RegionRestrictionSection
      defaultValues={{
        enabled: settings.RegionRestrictionEnabled,
        countries: settings.RegionRestrictionCountries,
        url: settings.RegionRestrictionRedirectURL,
      }}
    />
  ),
  'token-limits': (settings: SecuritySettings) => (
    <TokenLimitSection
      defaultValues={{
        'token_setting.max_user_tokens':
          settings['token_setting.max_user_tokens'],
      }}
    />
  ),
}

export type { SecuritySectionId } from '../section-navigation'

const securityRegistry = createSectionRegistry<
  SecuritySectionId,
  SecuritySettings
>({
  sections: SECURITY_SECTIONS.map((section) => ({
    ...section,
    build: SECURITY_SECTION_BUILDERS[section.id],
  })),
  defaultSection: 'rate-limit',
  basePath: '/system-settings/security',
  urlStyle: 'path',
})

export const SECURITY_SECTION_IDS = securityRegistry.sectionIds
export const SECURITY_DEFAULT_SECTION = securityRegistry.defaultSection
export const getSecuritySectionNavItems = securityRegistry.getSectionNavItems
export const getSecuritySectionContent = securityRegistry.getSectionContent
export const getSecuritySectionMeta = securityRegistry.getSectionMeta
