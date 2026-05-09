import { useTranslation } from 'react-i18next'
import { VideoIcon } from 'lucide-react'

export function VideoTab() {
  const { t } = useTranslation()

  return (
    <div className='flex flex-col items-center justify-center gap-4 py-20 text-center'>
      <VideoIcon className='text-muted-foreground size-12' />
      <p className='text-muted-foreground text-sm'>
        {t('Video generation is coming soon')}
      </p>
    </div>
  )
}
