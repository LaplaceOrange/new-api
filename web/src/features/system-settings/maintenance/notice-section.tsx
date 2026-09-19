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
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

function isHttpUrl(value: string) {
  try {
    const parsed = new URL(value)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:'
  } catch {
    return false
  }
}

const noticeShape = z.object({
  Notice: z.string().optional(),
  GroupChatLink: z.string().trim().optional(),
})

type NoticeFormValues = z.infer<typeof noticeShape>

type NoticeSectionProps = {
  defaultNotice: string
  defaultGroupChatLink: string
}

export function NoticeSection(props: NoticeSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const noticeSchema = useMemo(
    () =>
      noticeShape.extend({
        GroupChatLink: z
          .string()
          .trim()
          .refine((value) => value === '' || isHttpUrl(value), {
            message: t('Enter an http(s) group chat URL'),
          })
          .optional(),
      }),
    [t]
  )
  const form = useForm<NoticeFormValues>({
    resolver: zodResolver(noticeSchema),
    defaultValues: {
      Notice: props.defaultNotice ?? '',
      GroupChatLink: props.defaultGroupChatLink ?? '',
    },
  })

  useEffect(() => {
    form.reset({
      Notice: props.defaultNotice ?? '',
      GroupChatLink: props.defaultGroupChatLink ?? '',
    })
  }, [props.defaultNotice, props.defaultGroupChatLink, form])

  const onSubmit = async (values: NoticeFormValues) => {
    const notice = values.Notice ?? ''
    const groupChatLink = (values.GroupChatLink ?? '').trim()
    if (notice !== (props.defaultNotice ?? '')) {
      await updateOption.mutateAsync({
        key: 'Notice',
        value: notice,
      })
    }
    if (groupChatLink !== (props.defaultGroupChatLink ?? '').trim()) {
      await updateOption.mutateAsync({
        key: 'GroupChatLink',
        value: groupChatLink,
      })
    }
  }

  return (
    <SettingsSection title={t('System Notice')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save notice'
          />
          <FormField
            control={form.control}
            name='GroupChatLink'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Official group chat link')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder='https://t.me/your-group'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Used by the homepage Official Group Chat button. Leave empty to hide the button.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='Notice'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Announcement content')}</FormLabel>
                <FormControl>
                  <Textarea
                    rows={8}
                    placeholder={t(
                      'Planned maintenance on Friday at 22:00 UTC...'
                    )}
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
