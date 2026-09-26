import { createFileRoute } from '@tanstack/react-router'

import { Main } from '@/components/layout'
import { ImageStudio } from '@/features/image-studio'

export const Route = createFileRoute('/_authenticated/image-studio/')({
  component: () => <Main className='p-0'><ImageStudio /></Main>,
})
