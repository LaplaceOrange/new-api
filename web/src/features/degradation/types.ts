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
export type DegradationEvent = {
  status: string
  detected_model: string
  created_at: number
}

export type DegradationModel = {
  model: string
  sort: number
  status: string
  detected_model: string
  next_check_at: number
  timeline: DegradationEvent[]
}

export type DegradationGroup = {
  group: string
  sort: number
  models: DegradationModel[]
}

export type DegradationConfigModel = {
  model: string
  expected: string
  sort: number
}

export type DegradationConfigGroup = {
  group: string
  sort: number
  models: DegradationConfigModel[]
}

export type DegradationConfig = {
  enabled: boolean
  interval_minutes: number
  retry_count: number
  retry_interval_minutes: number
  groups: DegradationConfigGroup[]
}

export type DegradationPage = {
  success: boolean
  message?: string
  data: {
    groups: DegradationGroup[]
    config?: DegradationConfig
  }
}
