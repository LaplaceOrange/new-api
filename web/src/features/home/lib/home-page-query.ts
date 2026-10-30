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
import { queryOptions } from '@tanstack/react-query'

import { requireServerSuccess } from '@/lib/server-error-message'

import { getHomePageContent } from '../api'

const STORAGE_KEY = 'home_page_content'

export function readCachedHomePageContent(): string | undefined {
  try {
    return localStorage.getItem(STORAGE_KEY) ?? undefined
  } catch {
    return undefined
  }
}

export const homePageContentQueryOptions = queryOptions({
  queryKey: ['home-page-content'],
  queryFn: async () => {
    const response = requireServerSuccess(await getHomePageContent())
    const content = response.data ?? ''
    try {
      localStorage.setItem(STORAGE_KEY, content)
    } catch {
      // The network result remains usable when browser storage is unavailable.
    }
    return content
  },
  staleTime: 10 * 1000,
  meta: { errorToast: false },
})
