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
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

const arkAssetSchema = z.object({
  ArkAssetAccessKey: z.string(),
  ArkAssetSecretKey: z.string(),
  ArkAssetRegion: z.string(),
  ArkAssetProjectName: z.string(),
  ArkAssetGroupId: z.string(),
})

type ArkAssetFormValues = z.infer<typeof arkAssetSchema>

type ArkAssetSettingsSectionProps = {
  defaultValues: ArkAssetFormValues
}

export function ArkAssetSettingsSection({ defaultValues }: ArkAssetSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<ArkAssetFormValues>({
    resolver: zodResolver(arkAssetSchema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  const onSubmit = async (values: ArkAssetFormValues) => {
    const keys: (keyof ArkAssetFormValues)[] = [
      'ArkAssetAccessKey',
      'ArkAssetSecretKey',
      'ArkAssetRegion',
      'ArkAssetProjectName',
      'ArkAssetGroupId',
    ]

    for (const key of keys) {
      if (values[key] !== defaultValues[key]) {
        await updateOption.mutateAsync({ key, value: values[key] })
      }
    }
  }

  return (
    <SettingsSection
      title={t('Ark Asset Settings')}
      description={t('Configure Volcengine Ark Asset API credentials for Seedance resource uploads')}
    >
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
          <FormField
            control={form.control}
            name='ArkAssetAccessKey'
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
            name='ArkAssetSecretKey'
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
            name='ArkAssetRegion'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Region')}</FormLabel>
                <FormControl>
                  <Input {...field} placeholder='cn-beijing' />
                </FormControl>
                <FormDescription>
                  {t('Volcengine region, defaults to cn-beijing')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='ArkAssetProjectName'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Project Name')}</FormLabel>
                <FormControl>
                  <Input {...field} placeholder='default' />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='ArkAssetGroupId'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Asset Group ID')}</FormLabel>
                <FormControl>
                  <Input {...field} placeholder='' />
                </FormControl>
                <FormDescription>
                  {t('Leave empty to use the default asset group')}
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
