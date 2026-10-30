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
import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { isHttpUrl } from '@/lib/content-format'
import { handleServerError } from '@/lib/handle-server-error'

import {
  homePageContentQueryOptions,
  readCachedHomePageContent,
} from '../lib/home-page-query'
import type { HomePageContentResult } from '../types'

/**
 * Hook to load and manage custom home page content
 * Supports both Markdown/HTML content and iframe URLs
 */
export function useHomePageContent(): HomePageContentResult {
  const { t } = useTranslation()
  const query = useQuery({
    ...homePageContentQueryOptions,
    placeholderData: readCachedHomePageContent,
  })
  const content = query.data ?? readCachedHomePageContent() ?? ''

  useEffect(() => {
    if (query.error) {
      handleServerError(query.error, t('Failed to load home page content'))
    }
  }, [query.error, t])

  const isUrl = isHttpUrl(content)

  return { content, isLoaded: !query.isPending, isUrl }
}
