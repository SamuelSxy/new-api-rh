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
import { Construction } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Markdown } from '@/components/ui/markdown'
import { Skeleton } from '@/components/ui/skeleton'
import { PublicLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'
import { getAboutContent } from './api'

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

function EmptyAboutState() {
  const { t } = useTranslation()
  const currentYear = new Date().getFullYear()

  return (
    <div className='py-16 text-center'>
      <div className='mx-auto max-w-2xl space-y-6'>
        <div className='flex justify-center'>
          <div className='flex size-20 items-center justify-center rounded-2xl border border-blue-500/20 bg-blue-500/5 dark:border-blue-400/20 dark:bg-blue-400/5'>
            <Construction className='h-10 w-10 text-blue-500 dark:text-blue-400' />
          </div>
        </div>
        <div className='space-y-3'>
          <div className='inline-flex items-center gap-1.5 rounded-full border border-blue-500/20 bg-blue-500/5 px-3 py-1.5 text-[11px] font-medium text-blue-600 shadow-xs dark:border-blue-400/20 dark:bg-blue-400/5 dark:text-blue-400'>
            <span className='relative flex size-1.5'>
              <span className='absolute inline-flex h-full w-full animate-ping rounded-full bg-blue-400 opacity-75' />
              <span className='relative inline-flex size-1.5 rounded-full bg-blue-500 dark:bg-blue-400' />
            </span>
            <span>{t('About')}</span>
          </div>
          <h2 className='text-[clamp(1.5rem,3vw,2rem)] font-bold tracking-tight'>{t('No About Content Set')}</h2>
          <p className='text-muted-foreground'>
            {t(
              'The administrator has not configured any about content yet. You can set it in the settings page, supporting HTML or URL.'
            )}
          </p>
        </div>
        <div className='space-y-4 text-sm'>
          <p>
            {t('New API Project Repository:')}{' '}
            <a
              href='https://github.com/QuantumNous/new-api'
              target='_blank'
              rel='noopener noreferrer'
              className='text-primary hover:underline'
            >
              {t('https://github.com/QuantumNous/new-api')}
            </a>
          </p>
          <p className='text-muted-foreground'>
            <a
              href='https://github.com/QuantumNous/new-api'
              target='_blank'
              rel='noopener noreferrer'
              className='text-primary hover:underline'
            >
              {t('NewAPI')}
            </a>{' '}
            © {currentYear}{' '}
            <a
              href='https://github.com/QuantumNous'
              target='_blank'
              rel='noopener noreferrer'
              className='text-primary hover:underline'
            >
              {t('QuantumNous')}
            </a>{' '}
            {t('| Based on')}{' '}
            <a
              href='https://github.com/songquanpeng/one-api'
              target='_blank'
              rel='noopener noreferrer'
              className='text-primary hover:underline'
            >
              {t('One API')}
            </a>{' '}
            © 2023{' '}
            <a
              href='https://github.com/songquanpeng'
              target='_blank'
              rel='noopener noreferrer'
              className='text-primary hover:underline'
            >
              {t('JustSong')}
            </a>
          </p>
          <p className='text-muted-foreground'>
            {t('This project must be used in compliance with the')}{' '}
            <a
              href='https://github.com/QuantumNous/new-api/blob/main/LICENSE'
              target='_blank'
              rel='noopener noreferrer'
              className='text-primary hover:underline'
            >
              {t('AGPL v3.0 License')}
            </a>
            .
          </p>
        </div>
      </div>
    </div>
  )
}

export function About() {
  const { t } = useTranslation()
  const { data, isLoading } = useQuery({
    queryKey: ['about-content'],
    queryFn: getAboutContent,
  })

  const rawContent = data?.data?.trim() ?? ''
  const hasContent = rawContent.length > 0
  const isUrl = hasContent && isValidUrl(rawContent)
  const isHtml = hasContent && !isUrl && isLikelyHtml(rawContent)

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

  if (!hasContent) {
    return (
      <PublicLayout showMainContainer={false}>
        <div className='relative'>
          <GradientBackground />
          <PageTransition className='relative mx-auto w-full max-w-4xl px-3 pt-16 pb-8 sm:px-6 sm:pt-20 xl:px-8'>
            <EmptyAboutState />
          </PageTransition>
        </div>
      </PublicLayout>
    )
  }

  if (isUrl) {
    return (
      <PublicLayout showMainContainer={false}>
        <iframe
          src={rawContent}
          className='h-[calc(100vh-3.5rem)] w-full border-0'
          title={t('About')}
        />
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <div className='relative'>
        <GradientBackground />
        <PageTransition className='relative mx-auto w-full max-w-4xl px-3 pt-16 pb-8 sm:px-6 sm:pt-20 xl:px-8'>
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
