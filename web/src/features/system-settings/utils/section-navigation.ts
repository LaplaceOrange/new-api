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
import type { TFunction } from 'i18next'

export type SectionNavigationConfig<TSectionId extends string> = {
  sections: readonly { id: TSectionId; titleKey: string }[]
  defaultSection: TSectionId
  basePath: string
  urlStyle?: 'query' | 'path'
}

export function createSectionNavigation<TSectionId extends string>(
  config: SectionNavigationConfig<TSectionId>
) {
  return {
    sectionIds: config.sections.map((section) => section.id) as [
      TSectionId,
      ...TSectionId[],
    ],
    defaultSection: config.defaultSection,
    getSectionNavItems: (t: TFunction) =>
      config.sections.map((section) => ({
        title: t(section.titleKey),
        url:
          config.urlStyle === 'path'
            ? `${config.basePath}/${section.id}`
            : `${config.basePath}?section=${section.id}`,
      })),
  }
}
