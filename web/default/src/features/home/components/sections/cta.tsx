import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { AnimateInView } from '@/components/animate-in-view'

interface CTAProps {
  className?: string
  isAuthenticated?: boolean
}

export function CTA(props: CTAProps) {
  const { t } = useTranslation()

  if (props.isAuthenticated) {
    return null
  }

  return (
    <section className='relative z-10 overflow-hidden bg-[#171717] px-6 py-24 md:py-32'>
      {/* Gradient mesh background */}
      <div
        aria-hidden
        className='absolute inset-0 -z-10 opacity-20 dark:opacity-[0.08]'
        style={{
          background: [
            'radial-gradient(ellipse 50% 50% at 30% 50%, oklch(0.7 0.15 250 / 70%) 0%, transparent 70%)',
            'radial-gradient(ellipse 40% 40% at 70% 40%, oklch(0.65 0.12 200 / 50%) 0%, transparent 70%)',
          ].join(', '),
        }}
      />

      <AnimateInView
        className='mx-auto max-w-2xl text-center'
        animation='scale-in'
      >
        <h2 className='text-2xl leading-tight font-bold tracking-tight text-white md:text-4xl'>
          {t('Ready to simplify')}
          <br />
          <span className='bg-gradient-to-b from-[#80FF00] to-[#FBFF00] bg-clip-text text-transparent'>
            {t('your AI integration?')}
          </span>
        </h2>
        <p className='mx-auto mt-5 max-w-md text-sm leading-relaxed text-[#808080] md:text-base'>
          {t(
            'Deploy your own gateway and start routing requests through your configured upstream services.'
          )}
        </p>
        <div className='mt-8 flex items-center justify-center gap-4'>
          <Link
            to='/sign-up'
            className='inline-flex items-center rounded-full bg-gradient-to-b from-[#80FF00] to-[#FBFF00] px-8 py-3 text-sm font-semibold text-black transition-opacity hover:opacity-90'
          >
            {t('Get Started')}
          </Link>
          <Link
            to='/pricing'
            className='inline-flex items-center rounded-full border border-white/10 bg-white/[0.05] px-6 py-3 text-sm text-white transition-colors hover:bg-white/[0.08]'
          >
            {t('View Pricing')}
          </Link>
        </div>
      </AnimateInView>
    </section>
  )
}
