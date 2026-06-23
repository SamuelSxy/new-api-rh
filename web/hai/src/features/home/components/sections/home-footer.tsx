import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useSystemConfig } from '@/hooks/use-system-config'
import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

// Social icon SVGs (inline, no external dependency)
const FacebookIcon = () => (
  <svg viewBox='0 0 24 24' fill='currentColor' className='size-5'>
    <path d='M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z' />
  </svg>
)
const YoutubeIcon = () => (
  <svg viewBox='0 0 24 24' fill='currentColor' className='size-5'>
    <path d='M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z' />
  </svg>
)
const InstagramIcon = () => (
  <svg viewBox='0 0 24 24' fill='currentColor' className='size-5'>
    <path d='M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zM12 0C8.741 0 8.333.014 7.053.072 2.695.272.273 2.69.073 7.052.014 8.333 0 8.741 0 12c0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98C8.333 23.986 8.741 24 12 24c3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98C15.668.014 15.259 0 12 0zm0 5.838a6.162 6.162 0 1 0 0 12.324 6.162 6.162 0 0 0 0-12.324zM12 16a4 4 0 1 1 0-8 4 4 0 0 1 0 8zm6.406-11.845a1.44 1.44 0 1 0 0 2.881 1.44 1.44 0 0 0 0-2.881z' />
  </svg>
)
const TwitterIcon = () => (
  <svg viewBox='0 0 24 24' fill='currentColor' className='size-5'>
    <path d='M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-4.714-6.231-5.401 6.231H2.747l7.73-8.835L1.254 2.25H8.08l4.253 5.622zm-1.161 17.52h1.833L7.084 4.126H5.117z' />
  </svg>
)
const WhatsappIcon = () => (
  <svg viewBox='0 0 24 24' fill='currentColor' className='size-5'>
    <path d='M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 0 1-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 0 1-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 0 1 2.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0 0 12.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 0 0 5.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 0 0-3.48-8.413z' />
  </svg>
)

const NAV_LINKS = [
  { text: '主页', href: '/' as const },
  { text: '控制台', href: '/dashboard' as const },
  { text: '模型广场', href: '/pricing' as const },
  { text: '排行榜', href: '/rankings' as const },
  { text: '文档', href: '/docs' as const },
  { text: '关于', href: '/about' as const },
]

export function HomeFooter() {
  const { systemName, logo: systemLogo } = useSystemConfig()
  const topNavLinks = useTopNavLinks()
  const [comingSoonOpen, setComingSoonOpen] = useState(false)
  const currentYear = new Date().getFullYear()
  const displayName = systemName || 'New API'

  // Use dynamic nav links from top nav (including comingSoon items)
  const navLinks = topNavLinks.length > 0 ? topNavLinks : NAV_LINKS

  return (
    <>
    <footer className='overflow-hidden rounded-t-3xl' style={{ background: '#272727' }}>
      <div className='px-6 py-16 md:px-20 lg:px-24'>
        {/* Main row */}
        <div className='flex flex-col justify-between gap-12 md:flex-row'>
          {/* Brand column */}
          <div className='flex flex-col items-start gap-6'>
            {/* Logo image only */}
            {systemLogo ? (
              <img
                src={systemLogo}
                alt={displayName}
                className='h-12 w-auto object-contain self-start'
              />
            ) : (
              <span className='text-2xl font-bold text-white'>{displayName}</span>
            )}

            {/* Brand description */}
            <p className='max-w-xs text-base leading-relaxed text-white/40'>
              专业的 AIGC 内容生成平台，提供高质量 AI 视频、图文生成服务，官方授权，价格稳定。
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

            {/* Social icons */}
            <div className='flex items-center gap-3'>
              {[FacebookIcon, YoutubeIcon, WhatsappIcon, InstagramIcon, TwitterIcon].map(
                (Icon, i) => (
                  <button
                    key={i}
                    className='flex size-8 items-center justify-center rounded-full text-white/50 transition-colors hover:text-white'
                    style={{ background: 'rgba(255,255,255,0.05)' }}
                  >
                    <Icon />
                  </button>
                )
              )}
            </div>
          </div>

          {/* Right: nav columns */}
          <div className='flex gap-16'>
            {/* Navigation */}
            <div className='flex flex-col gap-6'>
              <h4 className='text-xl font-bold' style={{ color: '#80FF00' }}>
                导航
              </h4>
              <div className='flex flex-col gap-4'>
                {navLinks.map((link) => (
                  link.comingSoon ? (
                    <button
                      key={link.title}
                      onClick={() => setComingSoonOpen(true)}
                      className='text-left text-base text-white/70 transition-colors hover:text-white'
                    >
                      {link.title}
                    </button>
                  ) : link.external ? (
                    <a
                      key={link.title}
                      href={link.href}
                      target='_blank'
                      rel='noopener noreferrer'
                      className='text-base text-white/70 transition-colors hover:text-white'
                    >
                      {link.title}
                    </a>
                  ) : (
                    <Link
                      key={link.title}
                      to={link.href as never}
                      className='text-base text-white/70 transition-colors hover:text-white'
                    >
                      {link.title}
                    </Link>
                  )
                ))}
              </div>
            </div>

            {/* Contact */}
            <div className='flex flex-col gap-6'>
              <h4 className='text-xl font-bold' style={{ color: '#80FF00' }}>
                联系我们
              </h4>
              <div className='flex flex-col gap-4'>
                <a
                  href='mailto:wangdaoyu@shybsd.cn'
                  className='text-base text-white/70 transition-colors hover:text-white'
                >
                  wangdaoyu@shybsd.cn
                </a>
                <Link
                  to='/privacy-policy'
                  className='text-base text-white/70 transition-colors hover:text-white'
                >
                  隐私政策
                </Link>
                <Link
                  to='/user-agreement'
                  className='text-base text-white/70 transition-colors hover:text-white'
                >
                  服务协议
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Divider */}
      <div className='mx-6 border-t border-white/10 md:mx-20 lg:mx-24' />

      {/* Bottom bar */}
      <div className='flex flex-col items-start justify-between gap-3 px-6 py-5 text-sm sm:flex-row sm:items-center md:px-20 lg:px-24'>
        <p className='text-white/40'>
          Copyright© {currentYear} {displayName} All Rights Reserved.
        </p>
        <div className='flex items-center gap-4 text-white/40'>
          <a
            href='https://beian.miit.gov.cn'
            target='_blank'
            rel='noopener noreferrer'
            className='transition-colors hover:text-white/70'
          >
            平台资质
          </a>
          <span className='text-white/20'>|</span>
          <Link to='/user-agreement' className='transition-colors hover:text-white/70'>
            服务协议
          </Link>
        </div>
      </div>
    </footer>

    {/* Coming Soon dialog */}
    <Dialog open={comingSoonOpen} onOpenChange={setComingSoonOpen}>
      <DialogContent className='sm:max-w-sm'>
        <DialogHeader>
          <DialogTitle>AIGC社群</DialogTitle>
          <DialogDescription>该功能正在紧张开发中，敬请期待！</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button onClick={() => setComingSoonOpen(false)}>我知道了</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
    </>
  )
}
