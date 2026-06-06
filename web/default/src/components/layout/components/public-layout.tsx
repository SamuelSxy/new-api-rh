import type { TopNavLink } from '../types'
import { PublicHeader, type PublicHeaderProps } from './public-header'

function IcpFooter() {
  return (
    <footer className='border-t border-border/40 py-4 text-center'>
      <a
        href='https://beian.miit.gov.cn'
        target='_blank'
        rel='noopener noreferrer'
        className='text-muted-foreground/60 hover:text-muted-foreground text-xs transition-colors'
      >
        鲁ICP备2026028251号-1
      </a>
    </footer>
  )
}

type PublicLayoutProps = {
  children: React.ReactNode
  showMainContainer?: boolean
  navContent?: React.ReactNode
  headerProps?: Omit<PublicHeaderProps, 'navContent'>
  navLinks?: TopNavLink[]
  showThemeSwitch?: boolean
  showAuthButtons?: boolean
  showNotifications?: boolean
  logo?: React.ReactNode
  siteName?: string
}

export function PublicLayout(props: PublicLayoutProps) {
  return (
    <div className='bg-background text-foreground relative min-h-svh overflow-hidden'>
      <PublicHeader
        navContent={props.navContent}
        navLinks={props.navLinks}
        showThemeSwitch={props.showThemeSwitch}
        showAuthButtons={props.showAuthButtons}
        showNotifications={props.showNotifications}
        logo={props.logo}
        siteName={props.siteName}
        {...props.headerProps}
      />

      {props.showMainContainer !== false ? (
        <>
          <main className='container px-4 py-6 pt-20 md:px-4'>
            {props.children}
          </main>
          <IcpFooter />
        </>
      ) : (
        <>
          {props.children}
          <IcpFooter />
        </>
      )}
    </div>
  )
}
