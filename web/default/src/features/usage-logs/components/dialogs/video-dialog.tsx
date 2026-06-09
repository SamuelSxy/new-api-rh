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
import { useState } from 'react'
import { ExternalLink, Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'

interface VideoDialogProps {
  videoUrl: string
  taskId?: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function VideoDialog({
  videoUrl,
  taskId,
  open,
  onOpenChange,
}: VideoDialogProps) {
  const { t } = useTranslation()
  const [isLoading, setIsLoading] = useState(true)
  const [hasError, setHasError] = useState(false)

  const handleOpenChange = (newOpen: boolean) => {
    if (newOpen) {
      setIsLoading(true)
      setHasError(false)
    }
    onOpenChange(newOpen)
  }

  const handleVideoLoaded = () => {
    setIsLoading(false)
    setHasError(false)
  }

  const handleVideoError = () => {
    setIsLoading(false)
    setHasError(true)
  }

  const handleCopy = () => {
    navigator.clipboard.writeText(videoUrl).then(() => {
      toast.success(t('Copied to clipboard'))
    })
  }

  const handleOpenInNewTab = () => {
    window.open(videoUrl, '_blank', 'noopener,noreferrer')
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className='sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle>{t('Video Preview')}</DialogTitle>
          <DialogDescription>
            {taskId
              ? `${t('Task ID:')} ${taskId}`
              : t('View the generated video')}
          </DialogDescription>
        </DialogHeader>

        <div className='py-2'>
          <div className='bg-muted/50 relative flex min-h-[300px] items-center justify-center rounded-lg border'>
            {isLoading && !hasError && (
              <Skeleton className='absolute inset-0 h-full w-full rounded-lg' />
            )}

            {!hasError ? (
              <video
                src={videoUrl}
                controls
                className={`max-h-[500px] w-full rounded-lg ${isLoading ? 'opacity-0' : 'opacity-100'}`}
                onLoadedData={handleVideoLoaded}
                onError={handleVideoError}
              />
            ) : (
              <div className='flex flex-col items-center gap-3 p-8 text-center'>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Video cannot be played in this browser. This may be due to cross-origin restrictions, authentication requirements, or hotlink protection.'
                  )}
                </p>
                <div className='flex gap-2'>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={handleOpenInNewTab}
                  >
                    <ExternalLink className='mr-1.5 size-3.5' />
                    {t('Open in new tab')}
                  </Button>
                  <Button variant='outline' size='sm' onClick={handleCopy}>
                    <Copy className='mr-1.5 size-3.5' />
                    {t('Copy URL')}
                  </Button>
                </div>
              </div>
            )}
          </div>

          {!hasError && !isLoading && (
            <div className='mt-2 flex justify-end gap-2'>
              <Button variant='ghost' size='sm' onClick={handleCopy}>
                <Copy className='mr-1.5 size-3.5' />
                {t('Copy URL')}
              </Button>
              <Button variant='ghost' size='sm' onClick={handleOpenInNewTab}>
                <ExternalLink className='mr-1.5 size-3.5' />
                {t('Open in new tab')}
              </Button>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
