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

const tosSchema = z.object({
  TosEnabled: z.boolean(),
  TosAccessKey: z.string(),
  TosSecretKey: z.string(),
  TosRegion: z.string(),
  TosBucket: z.string(),
  TosEndpoint: z.string(),
  TosPublicRead: z.boolean(),
  TosCustomDomain: z.string(),
})

type TosFormValues = z.infer<typeof tosSchema>

type TosSettingsSectionProps = {
  defaultValues: TosFormValues
}

export function TosSettingsSection({ defaultValues }: TosSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<TosFormValues>({
    resolver: zodResolver(tosSchema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  const onSubmit = async (values: TosFormValues) => {
    const keys: (keyof TosFormValues)[] = [
      'TosEnabled',
      'TosAccessKey',
      'TosSecretKey',
      'TosRegion',
      'TosBucket',
      'TosEndpoint',
      'TosPublicRead',
      'TosCustomDomain',
    ]

    for (const key of keys) {
      if (values[key] !== defaultValues[key]) {
        await updateOption.mutateAsync({ key, value: String(values[key]) })
      }
    }
  }

  return (
    <SettingsSection
      title={t('TOS Object Storage')}
      description={t('Configure Volcengine TOS object storage for studio asset uploads')}
    >
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
          <FormField
            control={form.control}
            name='TosEnabled'
            render={({ field }) => (
              <FormItem className='flex items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-base'>{t('Enable TOS')}</FormLabel>
                  <FormDescription>
                    {t('Upload studio assets to Volcengine TOS instead of local disk')}
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
            name='TosAccessKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Access Key')}</FormLabel>
                <FormControl>
                  <Input {...field} placeholder='AK...' autoComplete='off' />
                </FormControl>
                <FormDescription>
                  {t('Volcengine IAM Access Key ID')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='TosSecretKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Secret Key')}</FormLabel>
                <FormControl>
                  <Input {...field} type='password' placeholder='SK...' autoComplete='new-password' />
                </FormControl>
                <FormDescription>
                  {t('Volcengine IAM Secret Access Key')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='TosRegion'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Region')}</FormLabel>
                <FormControl>
                  <Input {...field} placeholder='cn-beijing' />
                </FormControl>
                <FormDescription>
                  {t('TOS region, defaults to cn-beijing')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='TosBucket'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Bucket')}</FormLabel>
                <FormControl>
                  <Input {...field} placeholder='my-bucket' />
                </FormControl>
                <FormDescription>
                  {t('TOS bucket name')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='TosEndpoint'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Endpoint')}</FormLabel>
                <FormControl>
                  <Input {...field} placeholder='tos-cn-beijing.volces.com' />
                </FormControl>
                <FormDescription>
                  {t('TOS endpoint URL')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='TosPublicRead'
            render={({ field }) => (
              <FormItem className='flex items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-base'>{t('Public Read')}</FormLabel>
                  <FormDescription>
                    {t('Upload objects with public-read ACL so upstream model services (Ark, Seedance, etc.) can access files directly via URL. Recommended when your bucket does not have a public read policy.')}
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
            name='TosCustomDomain'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Custom Domain')}</FormLabel>
                <FormControl>
                  <Input {...field} placeholder='https://assets.example.com' />
                </FormControl>
                <FormDescription>
                  {t('Optional CDN or custom domain used to build file URLs (e.g. https://assets.example.com). Leave empty to use the default TOS bucket endpoint.')}
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
