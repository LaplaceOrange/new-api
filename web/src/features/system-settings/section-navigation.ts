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
import { createSectionNavigation } from './utils/section-navigation'

export const SITE_SECTIONS = [
  { id: 'system-info', titleKey: 'System Information' },
  { id: 'notice', titleKey: 'System Notice' },
  { id: 'header-navigation', titleKey: 'Header navigation' },
  { id: 'sidebar-modules', titleKey: 'Sidebar modules' },
] as const

export type SiteSectionId = (typeof SITE_SECTIONS)[number]['id']

const siteNavigation = createSectionNavigation<SiteSectionId>({
  sections: SITE_SECTIONS,
  defaultSection: 'system-info',
  basePath: '/system-settings/site',
  urlStyle: 'path',
})

export const SITE_SECTION_IDS = siteNavigation.sectionIds
export const SITE_DEFAULT_SECTION = siteNavigation.defaultSection
export const getSiteSectionNavItems = siteNavigation.getSectionNavItems

export const AUTH_SECTIONS = [
  { id: 'basic-auth', titleKey: 'Basic Authentication' },
  { id: 'oauth', titleKey: 'OAuth Integrations' },
  { id: 'passkey', titleKey: 'Passkey Authentication' },
  { id: 'bot-protection', titleKey: 'Bot Protection' },
  { id: 'custom-oauth', titleKey: 'Custom OAuth' },
] as const

export type AuthSectionId = (typeof AUTH_SECTIONS)[number]['id']

const authNavigation = createSectionNavigation<AuthSectionId>({
  sections: AUTH_SECTIONS,
  defaultSection: 'basic-auth',
  basePath: '/system-settings/auth',
  urlStyle: 'path',
})

export const AUTH_SECTION_IDS = authNavigation.sectionIds
export const AUTH_DEFAULT_SECTION = authNavigation.defaultSection
export const getAuthSectionNavItems = authNavigation.getSectionNavItems

export const BILLING_SECTIONS = [
  { id: 'quota', titleKey: 'Quota Settings' },
  { id: 'currency', titleKey: 'Currency & Display' },
  { id: 'model-pricing', titleKey: 'Model Pricing' },
  { id: 'group-pricing', titleKey: 'Group Pricing' },
  { id: 'payment', titleKey: 'Payment Gateway' },
  { id: 'checkin', titleKey: 'Check-in Rewards' },
] as const

export type BillingSectionId = (typeof BILLING_SECTIONS)[number]['id']

const billingNavigation = createSectionNavigation<BillingSectionId>({
  sections: BILLING_SECTIONS,
  defaultSection: 'quota',
  basePath: '/system-settings/billing',
  urlStyle: 'path',
})

export const BILLING_SECTION_IDS = billingNavigation.sectionIds
export const BILLING_DEFAULT_SECTION = billingNavigation.defaultSection
export const getBillingSectionNavItems = billingNavigation.getSectionNavItems

export const MODELS_SECTIONS = [
  { id: 'global', titleKey: 'Global Model Configuration' },
  { id: 'routing-reliability', titleKey: 'Routing Reliability' },
  { id: 'gemini', titleKey: 'Gemini' },
  { id: 'claude', titleKey: 'Claude' },
  { id: 'grok', titleKey: 'Grok' },
  { id: 'channel-affinity', titleKey: 'Channel Affinity' },
  { id: 'model-deployment', titleKey: 'Model Deployment' },
] as const

export type ModelSectionId = (typeof MODELS_SECTIONS)[number]['id']

const modelsNavigation = createSectionNavigation<ModelSectionId>({
  sections: MODELS_SECTIONS,
  defaultSection: 'global',
  basePath: '/system-settings/models',
  urlStyle: 'path',
})

export const MODELS_SECTION_IDS = modelsNavigation.sectionIds
export const MODELS_DEFAULT_SECTION = modelsNavigation.defaultSection
export const getModelsSectionNavItems = modelsNavigation.getSectionNavItems

export const SECURITY_SECTIONS = [
  { id: 'rate-limit', titleKey: 'Rate Limiting' },
  { id: 'sensitive-words', titleKey: 'Sensitive Words' },
  { id: 'ssrf', titleKey: 'SSRF Protection' },
  { id: 'region-restriction', titleKey: 'Region Restriction' },
  { id: 'token-limits', titleKey: 'Token Limits' },
] as const

export type SecuritySectionId = (typeof SECURITY_SECTIONS)[number]['id']

const securityNavigation = createSectionNavigation<SecuritySectionId>({
  sections: SECURITY_SECTIONS,
  defaultSection: 'rate-limit',
  basePath: '/system-settings/security',
  urlStyle: 'path',
})

export const SECURITY_SECTION_IDS = securityNavigation.sectionIds
export const SECURITY_DEFAULT_SECTION = securityNavigation.defaultSection
export const getSecuritySectionNavItems = securityNavigation.getSectionNavItems

export const CONTENT_SECTIONS = [
  { id: 'dashboard', titleKey: 'Data Dashboard' },
  { id: 'announcements', titleKey: 'Announcements' },
  { id: 'api-info', titleKey: 'API Addresses' },
  { id: 'faq', titleKey: 'FAQ' },
  { id: 'uptime-kuma', titleKey: 'Uptime Kuma' },
  { id: 'chat', titleKey: 'Chat Presets' },
  { id: 'drawing', titleKey: 'Drawing' },
] as const

export type ContentSectionId = (typeof CONTENT_SECTIONS)[number]['id']

const contentNavigation = createSectionNavigation<ContentSectionId>({
  sections: CONTENT_SECTIONS,
  defaultSection: 'dashboard',
  basePath: '/system-settings/content',
  urlStyle: 'path',
})

export const CONTENT_SECTION_IDS = contentNavigation.sectionIds
export const CONTENT_DEFAULT_SECTION = contentNavigation.defaultSection
export const getContentSectionNavItems = contentNavigation.getSectionNavItems

export const OPERATIONS_SECTIONS = [
  { id: 'behavior', titleKey: 'System Behavior' },
  { id: 'alerts', titleKey: 'Monitoring & Alerts' },
  { id: 'email', titleKey: 'SMTP Email' },
  { id: 'worker', titleKey: 'Worker Proxy' },
  { id: 'logs', titleKey: 'Log Maintenance' },
  { id: 'performance', titleKey: 'Performance' },
  { id: 'update-checker', titleKey: 'System maintenance' },
] as const

export type OperationsSectionId = (typeof OPERATIONS_SECTIONS)[number]['id']

const operationsNavigation = createSectionNavigation<OperationsSectionId>({
  sections: OPERATIONS_SECTIONS,
  defaultSection: 'behavior',
  basePath: '/system-settings/operations',
  urlStyle: 'path',
})

export const OPERATIONS_SECTION_IDS = operationsNavigation.sectionIds
export const OPERATIONS_DEFAULT_SECTION = operationsNavigation.defaultSection
export const getOperationsSectionNavItems =
  operationsNavigation.getSectionNavItems
