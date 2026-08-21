import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { SettingsForm, SettingsSwitchContent, SettingsSwitchItem } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const schema = z.object({ enabled: z.boolean(), countries: z.string(), url: z.string().url() })
type Values = z.infer<typeof schema>
type Props = { defaultValues: Values }

export function RegionRestrictionSection({ defaultValues }: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues })
  useEffect(() => form.reset(defaultValues), [defaultValues, form])

  const onSubmit = async (values: Values) => {
    for (const [key, value] of Object.entries({
      RegionRestrictionEnabled: values.enabled,
      RegionRestrictionCountries: values.countries,
      RegionRestrictionRedirectURL: values.url,
    })) {
      const old = key === 'RegionRestrictionEnabled' ? defaultValues.enabled : key === 'RegionRestrictionCountries' ? defaultValues.countries : defaultValues.url
      if (value !== old) await updateOption.mutateAsync({ key, value })
    }
  }

  return <SettingsSection title={t('Region Restriction')}><Form {...form}><SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
    <SettingsPageFormActions onSave={form.handleSubmit(onSubmit)} isSaving={updateOption.isPending} saveLabel='Save region restriction' />
    <FormField control={form.control} name='enabled' render={({ field }) => <SettingsSwitchItem><SettingsSwitchContent><FormLabel>{t('Enable region restriction')}</FormLabel><FormDescription>{t('Visitors from listed countries will see the configured page instead of this site.')}</FormDescription></SettingsSwitchContent><FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl></SettingsSwitchItem>} />
    <FormField control={form.control} name='countries' render={({ field }) => <FormItem><FormLabel>{t('Restricted countries')}</FormLabel><FormControl><Input placeholder='CN,GB,US' {...field} /></FormControl><FormDescription>{t('Comma-separated two-letter country codes, for example CN,GB,US.')}</FormDescription><FormMessage /></FormItem>} />
    <FormField control={form.control} name='url' render={({ field }) => <FormItem><FormLabel>{t('Replacement page URL')}</FormLabel><FormControl><Input type='url' placeholder='https://example.com' {...field} /></FormControl><FormDescription>{t('The page fetched and displayed for restricted visitors.')}</FormDescription><FormMessage /></FormItem>} />
  </SettingsForm></Form></SettingsSection>
}
