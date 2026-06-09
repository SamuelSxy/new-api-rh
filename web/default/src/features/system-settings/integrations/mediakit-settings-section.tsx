import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
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
import { Switch } from '@/components/ui/switch'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

const mediakitSchema = z.object({
  MediakitEnabled: z.boolean(),
  MediakitApiKey: z.string(),
  MediakitToolVersion: z.string(),
  MediakitScene: z.string(),
  MediakitResolution: z.string(),
})

type MediakitFormValues = z.infer<typeof mediakitSchema>

type MediakitSettingsSectionProps = {
  defaultValues: MediakitFormValues
}

export function MediakitSettingsSection({ defaultValues }: MediakitSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<MediakitFormValues>({
    resolver: zodResolver(mediakitSchema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  const onSubmit = async (values: MediakitFormValues) => {
    const keys: (keyof MediakitFormValues)[] = [
      'MediakitEnabled',
      'MediakitApiKey',
      'MediakitToolVersion',
      'MediakitScene',
      'MediakitResolution',
    ]

    for (const key of keys) {
      if (values[key] !== defaultValues[key]) {
        await updateOption.mutateAsync({ key, value: String(values[key]) })
      }
    }
  }

  return (
    <SettingsSection
      title={t('Mediakit Video Enhancement')}
      description={t('Configure Volcengine Mediakit API for Seedance video enhancement')}
    >
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
          <FormField
            control={form.control}
            name='MediakitEnabled'
            render={({ field }) => (
              <FormItem className='flex items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-base'>{t('Enable Mediakit')}</FormLabel>
                  <FormDescription>
                    {t('Enable Mediakit video super-resolution for Seedance 2.0 fall models (480p → 720p/1080p)')}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='MediakitApiKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('API Key')}</FormLabel>
                <FormControl>
                  <Input {...field} type='password' placeholder='AK...' autoComplete='new-password' />
                </FormControl>
                <FormDescription>
                  {t('Volcengine IAM Access Key for Mediakit API')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='MediakitToolVersion'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Tool Version')}</FormLabel>
                <FormControl>
                  <Input {...field} placeholder='standard' />
                </FormControl>
                <FormDescription>
                  {t('Mediakit tool version, e.g. standard')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='MediakitScene'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Scene')}</FormLabel>
                <FormControl>
                  <Input {...field} placeholder='short_series' />
                </FormControl>
                <FormDescription>
                  {t('Mediakit scene parameter, e.g. short_series')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='MediakitResolution'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Default Resolution')}</FormLabel>
                <FormControl>
                  <Input {...field} placeholder='720p' />
                </FormControl>
                <FormDescription>
                  {t('Default target resolution when not specified by client (720p or 1080p)')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <Button type='submit' disabled={updateOption.isPending}>
            {t('Save')}
          </Button>
        </form>
      </Form>
    </SettingsSection>
  )
}
