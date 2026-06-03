import { Link } from '@tanstack/react-router'
import { Activity, Layers, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { HeroTerminalDemo } from '../hero-terminal-demo'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()

  return (
    <section className='relative overflow-hidden bg-[#171717] px-6 pt-28 pb-20 md:pt-36'>
      {/* Background glow */}
      <div
        aria-hidden
        className='pointer-events-none absolute inset-0 -z-10'
        style={{
          background: [
            'radial-gradient(ellipse 50% 40% at 15% 60%, rgba(128,255,0,0.05) 0%, transparent 60%)',
            'radial-gradient(ellipse 40% 30% at 75% 35%, rgba(0,200,209,0.05) 0%, transparent 60%)',
          ].join(', '),
        }}
      />

      <div className='mx-auto max-w-7xl'>
        {/* Two-column layout */}
        <div className='grid items-start gap-12 lg:grid-cols-2 lg:gap-16'>
          {/* Left: Feature highlights */}
          <div className='flex flex-col justify-center gap-10 lg:py-8'>
            {/* Feature 01 */}
            <div className='landing-animate-fade-up flex flex-col gap-3' style={{ animationDelay: '0ms' }}>
              <div className='flex items-center gap-3'>
                <div className='flex size-14 shrink-0 items-center justify-center rounded-xl bg-white/[0.06]'>
                  <Layers className='size-6 text-white' strokeWidth={1.5} />
                </div>
                <div className='flex flex-wrap items-baseline gap-2'>
                  <span className='text-2xl font-bold text-white md:text-3xl'>
                    {t('50+ Model Providers')}
                  </span>
                  <span className='text-2xl font-bold text-white/30 md:text-3xl'>
                    {t('No Markup')}
                  </span>
                </div>
              </div>
              <p className='pl-[68px] text-base text-[#808080] md:text-lg'>
                {t('Official channels · Stable long-term pricing')}
              </p>
            </div>

            {/* Feature 02 */}
            <div className='landing-animate-fade-up flex flex-col gap-3 opacity-0' style={{ animationDelay: '100ms' }}>
              <div className='flex items-center gap-3'>
                <div className='flex size-14 shrink-0 items-center justify-center rounded-xl bg-white/[0.06]'>
                  <Zap className='size-6 text-white' strokeWidth={1.5} />
                </div>
                <span className='text-2xl font-bold text-white md:text-3xl'>
                  {t('Unified API Management')}
                </span>
              </div>
              <div className='flex flex-wrap gap-3 pl-[68px]'>
                {['OpenAI', 'Claude', 'Gemini', 'DeepSeek', 'Qwen', 'Llama'].map((tag) => (
                  <span
                    key={tag}
                    className='rounded-full border border-white/10 bg-white/[0.05] px-5 py-1.5 text-sm text-white/70'
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </div>

            {/* Feature 03 */}
            <div className='landing-animate-fade-up flex flex-col gap-3 opacity-0' style={{ animationDelay: '200ms' }}>
              <div className='flex items-center gap-3'>
                <div className='flex size-14 shrink-0 items-center justify-center rounded-xl bg-white/[0.06]'>
                  <Activity className='size-6 text-white' strokeWidth={1.5} />
                </div>
                <span className='text-2xl font-bold text-white md:text-3xl'>
                  {t('High Concurrency')}
                </span>
              </div>
              <p className='pl-[68px] text-base text-[#808080] md:text-lg'>
                {t('Supports 1000+ concurrent requests · No waiting')}
              </p>
            </div>
          </div>

          {/* Right: API Demo */}
          <div
            className='landing-animate-fade-up overflow-hidden rounded-3xl opacity-0'
            style={{ animationDelay: '120ms' }}
          >
            <HeroTerminalDemo />
          </div>
        </div>

        {/* Action buttons */}
        <div
          className='landing-animate-fade-up mt-14 flex flex-wrap items-center gap-4 opacity-0'
          style={{ animationDelay: '300ms' }}
        >
          {props.isAuthenticated ? (
            <Link
              to='/dashboard'
              className='inline-flex items-center rounded-full bg-gradient-to-b from-[#80FF00] to-[#FBFF00] px-8 py-3 text-sm font-semibold text-black transition-opacity hover:opacity-90'
            >
              {t('Go to Dashboard')}
            </Link>
          ) : (
            <>
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
              <Link
                to='/about'
                className='inline-flex items-center rounded-full border border-white/10 bg-white/[0.05] px-6 py-3 text-sm text-white transition-colors hover:bg-white/[0.08]'
              >
                {t('About')}
              </Link>
            </>
          )}
        </div>
      </div>
    </section>
  )
}
