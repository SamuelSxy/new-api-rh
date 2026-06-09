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
import { useQuery } from '@tanstack/react-query'
import { FileWarning } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Markdown } from '@/components/ui/markdown'
import { Skeleton } from '@/components/ui/skeleton'
import { PublicLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'
import type { LegalDocumentResponse } from './types'

const gradientStyle = {
  background: [
    'radial-gradient(ellipse 60% 50% at 20% 20%, oklch(0.72 0.18 250 / 80%) 0%, transparent 70%)',
    'radial-gradient(ellipse 50% 40% at 80% 15%, oklch(0.65 0.15 200 / 60%) 0%, transparent 70%)',
    'radial-gradient(ellipse 40% 35% at 50% 70%, oklch(0.70 0.12 280 / 40%) 0%, transparent 70%)',
  ].join(', '),
  maskImage: 'linear-gradient(to bottom, black 40%, transparent 100%)',
  WebkitMaskImage: 'linear-gradient(to bottom, black 40%, transparent 100%)',
}

function GradientBackground() {
  return (
    <div
      aria-hidden
      className='pointer-events-none absolute inset-x-0 top-0 h-[500px] opacity-25 dark:opacity-[0.12]'
      style={gradientStyle}
    />
  )
}

function LegalPageHeader({ title, eyebrow }: { title: string; eyebrow: string }) {
  return (
    <header className='mb-10 space-y-4'>
      <div className='inline-flex items-center gap-1.5 rounded-full border border-blue-500/20 bg-blue-500/5 px-3 py-1.5 text-[11px] font-medium text-blue-600 shadow-xs dark:border-blue-400/20 dark:bg-blue-400/5 dark:text-blue-400'>
        <span className='relative flex size-1.5'>
          <span className='absolute inline-flex h-full w-full animate-ping rounded-full bg-blue-400 opacity-75' />
          <span className='relative inline-flex size-1.5 rounded-full bg-blue-500 dark:bg-blue-400' />
        </span>
        <span>{eyebrow}</span>
      </div>
      <h1 className='text-[clamp(1.75rem,4vw,2.5rem)] leading-[1.15] font-bold tracking-tight'>
        {title}
      </h1>
    </header>
  )
}

type LegalDocumentProps = {
  title: string
  queryKey: string
  fetchDocument: () => Promise<LegalDocumentResponse>
  emptyMessage: string
}

function isValidUrl(value: string) {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function isLikelyHtml(value: string) {
  return /<\/?[a-z][\s\S]*>/i.test(value)
}

export function LegalDocument({
  title,
  queryKey,
  fetchDocument,
  emptyMessage,
}: LegalDocumentProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useQuery({
    queryKey: [queryKey],
    queryFn: fetchDocument,
    staleTime: 10 * 60 * 1000,
  })

  const rawContent = data?.data?.trim() ?? ''
  const hasContent = rawContent.length > 0
  const isUrl = hasContent && isValidUrl(rawContent)
  const isHtml = hasContent && !isUrl && isLikelyHtml(rawContent)
  const success = data?.success ?? false

  if (isLoading) {
    return (
      <PublicLayout showMainContainer={false}>
        <div className='relative'>
          <GradientBackground />
          <div className='relative mx-auto w-full max-w-4xl px-3 pt-16 pb-8 sm:px-6 sm:pt-20 xl:px-8'>
            <div className='flex flex-col gap-4 py-8'>
              <Skeleton className='h-8 w-[45%]' />
              <Skeleton className='h-4 w-full' />
              <Skeleton className='h-4 w-[90%]' />
              <Skeleton className='h-4 w-[80%]' />
            </div>
          </div>
        </div>
      </PublicLayout>
    )
  }

  if (!success || !hasContent) {
    return (
      <PublicLayout showMainContainer={false}>
        <div className='relative'>
          <GradientBackground />
          <PageTransition className='relative mx-auto w-full max-w-4xl px-3 pt-16 pb-8 sm:px-6 sm:pt-20 xl:px-8'>
            <LegalPageHeader title={title} eyebrow={t('Legal')} />
            <Card className='border-dashed'>
              <CardHeader className='flex flex-row items-center gap-4'>
                <div className='bg-muted rounded-lg p-2'>
                  <FileWarning className='text-muted-foreground h-5 w-5' />
                </div>
                <div className='space-y-1'>
                  <CardTitle className='text-lg font-semibold'>{title}</CardTitle>
                  <p className='text-muted-foreground text-sm'>
                    {data?.message || emptyMessage}
                  </p>
                </div>
              </CardHeader>
            </Card>
          </PageTransition>
        </div>
      </PublicLayout>
    )
  }

  if (isUrl) {
    return (
      <PublicLayout showMainContainer={false}>
        <div className='relative'>
          <GradientBackground />
          <PageTransition className='relative mx-auto w-full max-w-4xl px-3 pt-16 pb-8 sm:px-6 sm:pt-20 xl:px-8'>
            <LegalPageHeader title={title} eyebrow={t('Legal')} />
            <Card>
              <CardContent className='space-y-4 pt-6'>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'The administrator configured an external link for this document.'
                  )}
                </p>
                <Button
                  render={
                    <a
                      href={rawContent}
                      target='_blank'
                      rel='noopener noreferrer'
                    />
                  }
                >
                  {t('View document')}
                </Button>
              </CardContent>
            </Card>
          </PageTransition>
        </div>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <div className='relative'>
        <GradientBackground />
        <PageTransition className='relative mx-auto w-full max-w-4xl px-3 pt-16 pb-8 sm:px-6 sm:pt-20 xl:px-8'>
          <LegalPageHeader title={title} eyebrow={t('Legal')} />

          {isHtml ? (
            <div
              className='prose prose-neutral dark:prose-invert max-w-none'
              dangerouslySetInnerHTML={{ __html: rawContent }}
            />
          ) : (
            <Markdown className='prose-neutral dark:prose-invert max-w-none'>
              {rawContent}
            </Markdown>
          )}
        </PageTransition>
      </div>
    </PublicLayout>
  )
}
