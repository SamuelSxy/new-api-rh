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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useSystemConfig } from '@/hooks/use-system-config'

type AuthLayoutProps = {
  children: React.ReactNode
}

export function AuthLayout({ children }: AuthLayoutProps) {
  const { t } = useTranslation()  
  const { logo, loading } = useSystemConfig()

  return (
    <div
      className='relative flex min-h-svh flex-col items-center justify-center overflow-hidden'
      style={{
        background: '#0a0a0a',
        backgroundImage: 'url(https://ai-gc.tos-cn-beijing.volces.com/asset/light.png)',
        backgroundSize: 'cover',
        backgroundPosition: 'center top',
        backgroundRepeat: 'no-repeat',
      }}
    >
      {/* Dark overlay */}
      <div className='pointer-events-none absolute inset-0 bg-[#0a0a0a]/65' />

      {/* Logo — top left, back to home */}
      <Link
        to='/'
        className='absolute top-6 left-6 z-10 flex items-center transition-opacity hover:opacity-80 sm:top-8 sm:left-8'
      >
        {loading ? (
          <div className='h-10 w-28 animate-pulse rounded-lg bg-white/10' />
        ) : (
          <img src={logo} alt={t('Logo')} className='h-10 w-auto object-contain' />
        )}
      </Link>

      {/* Frosted glass card */}
      <div
        className='relative z-10 w-full max-w-[460px] rounded-3xl p-8 sm:p-10'
        style={{
          background: 'rgba(255,255,255,0.06)',
          border: '1px solid rgba(255,255,255,0.12)',
          backdropFilter: 'blur(24px)',
          WebkitBackdropFilter: 'blur(24px)',
        }}
      >
        {children}
      </div>
    </div>
  )
}
