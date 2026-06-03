import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useSystemConfig } from '@/hooks/use-system-config'
import { HeaderLogo } from '@/components/layout/components/header-logo'

const NAV_LINKS = [
  { text: 'Home', href: '/' as const },
  { text: 'Pricing', href: '/pricing' as const },
  { text: 'About', href: '/about' as const },
]

const RESOURCE_LINKS = [
  { text: 'Documentation', href: 'https://docs.newapi.pro', external: true },
  { text: 'GitHub', href: 'https://github.com/QuantumNous/new-api', external: true },
  { text: 'Privacy Policy', href: '/privacy-policy' as const, external: false },
  { text: 'User Agreement', href: '/user-agreement' as const, external: false },
]

export function HomeFooter() {
  const { t } = useTranslation()
  const { systemName, logo: systemLogo, loading, logoLoaded } = useSystemConfig()
  const currentYear = new Date().getFullYear()
  const displayName = systemName || 'New API'

  return (
    <footer className='mx-4 mb-0 overflow-hidden rounded-t-3xl bg-[#111111] md:mx-8'>
      <div className='px-8 py-12 md:px-12'>
        <div className='grid gap-10 md:grid-cols-3'>
          {/* Brand column */}
          <div className='flex flex-col gap-5'>
            <div className='flex items-center gap-2.5'>
              <div className='flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-white/[0.08]'>
                <HeaderLogo
                  src={systemLogo}
                  loading={loading}
                  logoLoaded={logoLoaded}
                  className='size-full object-contain'
                />
              </div>
              <span className='text-sm font-semibold text-white'>{displayName}</span>
            </div>
            <p className='max-w-xs text-sm leading-relaxed text-white/35'>
              {t('Unified AI API gateway for developers and teams. Self-hosted, open-source, and extensible.')}
            </p>
            {/* New API attribution - protected */}
            <p className='text-xs text-white/20'>
              Powered by{' '}
              <a
                href='https://github.com/QuantumNous/new-api'
                target='_blank'
                rel='noopener noreferrer'
                className='text-white/35 transition-colors hover:text-white/60'
              >
                New API
              </a>
            </p>
          </div>

          {/* Navigation */}
          <div className='flex flex-col gap-5'>
            <h4 className='text-xs font-semibold tracking-widest text-white/40 uppercase'>
              {t('Navigation')}
            </h4>
            <div className='flex flex-col gap-3'>
              {NAV_LINKS.map((link) => (
                <Link
                  key={link.href}
                  to={link.href}
                  className='text-sm text-white/40 transition-colors hover:text-white/80'
                >
                  {t(link.text)}
                </Link>
              ))}
            </div>
          </div>

          {/* Resources */}
          <div className='flex flex-col gap-5'>
            <h4 className='text-xs font-semibold tracking-widest text-white/40 uppercase'>
              {t('Resources')}
            </h4>
            <div className='flex flex-col gap-3'>
              {RESOURCE_LINKS.map((link) =>
                link.external ? (
                  <a
                    key={link.href}
                    href={link.href}
                    target='_blank'
                    rel='noopener noreferrer'
                    className='text-sm text-white/40 transition-colors hover:text-white/80'
                  >
                    {t(link.text)}
                  </a>
                ) : (
                  <Link
                    key={link.href}
                    to={link.href as '/privacy-policy' | '/user-agreement'}
                    className='text-sm text-white/40 transition-colors hover:text-white/80'
                  >
                    {t(link.text)}
                  </Link>
                )
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Bottom bar */}
      <div className='border-t border-white/[0.05] px-8 py-4 md:px-12'>
        <div className='flex flex-col items-start justify-between gap-2 sm:flex-row sm:items-center'>
          <p className='text-xs text-white/25'>
            &copy; {currentYear} {displayName} {t('All Rights Reserved.')}
          </p>
          <div className='flex gap-4'>
            <Link
              to='/privacy-policy'
              className='text-xs text-white/25 transition-colors hover:text-white/50'
            >
              {t('Privacy Policy')}
            </Link>
            <Link
              to='/user-agreement'
              className='text-xs text-white/25 transition-colors hover:text-white/50'
            >
              {t('User Agreement')}
            </Link>
          </div>
        </div>
      </div>
    </footer>
  )
}
