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
type GroupRatioChangeProps = {
  ratio: number
  baseRatio?: number
}

export function GroupRatioChange(props: GroupRatioChangeProps) {
  if (props.baseRatio === undefined || props.baseRatio === props.ratio) {
    return <>{props.ratio}x</>
  }

  return (
    <span className='inline-flex items-center gap-1.5 whitespace-nowrap'>
      <del className='opacity-60'>{props.baseRatio}x</del>
      <span>{props.ratio}x</span>
    </span>
  )
}
