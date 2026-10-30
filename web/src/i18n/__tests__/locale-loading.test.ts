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
import { createInstance } from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { afterEach, expect, it } from 'vitest'

import { localeBackend } from '../backend'
import { convertDetectedLanguage } from '../languages'
import en from '../locales/en.json'

afterEach(() => {
  localStorage.clear()
})

it.each([
  ['zh-CN', 'zhCN', '加载中...'],
  ['zh-TW', 'zhTW', '載入中...'],
  ['fr-FR', 'fr', 'Chargement...'],
])(
  'loads the detected %s language before initialization resolves',
  async (detected, expectedLanguage, expectedText) => {
    localStorage.setItem('i18nextLng', detected)
    const i18n = createInstance()
    await i18n
      .use(localeBackend)
      .use(LanguageDetector)
      .init({
        resources: { en },
        partialBundledLanguages: true,
        fallbackLng: 'en',
        supportedLngs: ['en', 'zhCN', 'zhTW', 'fr', 'ru', 'ja', 'vi'],
        load: 'currentOnly',
        detection: {
          order: ['localStorage'],
          caches: [],
          convertDetectedLanguage,
        },
      })

    expect(i18n.resolvedLanguage).toBe(expectedLanguage)
    expect(i18n.t('Loading...')).toBe(expectedText)
  }
)

it('loads a newly selected language and retains English fallback', async () => {
  const i18n = createInstance()
  await i18n.use(localeBackend).init({
    lng: 'en',
    resources: { en },
    partialBundledLanguages: true,
    fallbackLng: 'en',
  })
  await i18n.changeLanguage('ja')
  expect(i18n.t('Loading...')).toBe('読み込み中...')
  expect(
    i18n.t('Missing translation', { defaultValue: 'English fallback' })
  ).toBe('English fallback')
  await i18n.changeLanguage('en')
  expect(i18n.t('Loading...')).toBe('Loading...')
})
