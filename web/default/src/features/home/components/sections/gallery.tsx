import { useEffect, useRef, useState } from 'react'
import { cn } from '@/lib/utils'
import { type GalleryVideo, GALLERY_COLUMNS, SHOW_CATEGORY_TABS } from '../../config/gallery.config'

/* ── Modal player ──────────────────────────────────────────── */
function VideoModal({ src, title, onClose }: { src: string; title: string; onClose: () => void }) {
  useEffect(() => {
    const h = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', h)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', h)
      document.body.style.overflow = ''
    }
  }, [onClose])

  return (
    <div
      className='fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm'
      onClick={onClose}
    >
      <div
        className='relative w-full max-w-3xl px-4'
        onClick={(e) => e.stopPropagation()}
      >
        {/* Close button */}
        <button
          onClick={onClose}
          className='absolute -top-10 right-4 flex size-8 items-center justify-center rounded-full bg-white/10 text-white transition-colors hover:bg-white/25'
          aria-label='关闭'
        >
          <svg viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth={2} className='size-4'>
            <path strokeLinecap='round' d='M18 6 6 18M6 6l12 12' />
          </svg>
        </button>

        <video
          src={src}
          autoPlay
          controls
          loop
          playsInline
          className='w-full rounded-2xl shadow-2xl'
        />
        <p className='mt-2.5 text-center text-sm text-white/50'>{title}</p>
      </div>
    </div>
  )
}

/* ── Card ──────────────────────────────────────────────────── */
function VideoCard({ title, src, poster, size = 'normal', id }: GalleryVideo) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const [modal, setModal] = useState(false)

  const isWide = size === 'wide'
  const aspect = size === 'tall' ? 'aspect-[9/16]' : 'aspect-video'
  const delay  = `${((id - 1) % 5) * 80}ms`

  const handleMouseEnter = () => { videoRef.current?.play().catch(() => {}) }
  const handleMouseLeave = () => {
    const v = videoRef.current
    if (!v) return
    v.pause()
    v.currentTime = 0
  }

  return (
    <>
      <div
        className={cn(
          'group cursor-pointer overflow-hidden rounded-2xl transition-all duration-300',
          'opacity-0 animate-[fadeInUp_0.5s_ease_forwards]',
          isWide
            ? 'border border-[rgba(128,255,0,0.18)] shadow-[0_0_18px_rgba(128,255,0,0.07)] hover:border-[rgba(128,255,0,0.4)] hover:shadow-[0_0_32px_rgba(128,255,0,0.18)]'
            : 'border border-white/[0.06] hover:border-white/25 hover:shadow-[0_8px_32px_rgba(0,0,0,0.5)]'
        )}
        style={{ background: '#1a1a1a', animationDelay: delay, animationFillMode: 'forwards' }}
        onClick={() => setModal(true)}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
      >
        <div className={cn('relative w-full overflow-hidden bg-black', aspect)}>
          <video
            ref={videoRef}
            src={src}
            poster={poster}
            loop
            muted
            playsInline
            preload='metadata'
            className='h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.03]'
          />

          {/* Bottom gradient + title */}
          <div className='pointer-events-none absolute inset-x-0 bottom-0 h-20 bg-gradient-to-t from-black/80 to-transparent' />
          <p className='pointer-events-none absolute bottom-2 left-3 right-3 truncate text-sm font-medium text-white/90 drop-shadow'>
            {title}
          </p>

        

          {/* Play icon — center, appears on hover */}
          <div
            className={cn(
              'pointer-events-none absolute inset-0 flex items-center justify-center',
              'opacity-0 transition-opacity duration-300 group-hover:opacity-100'
            )}
          >
            <div
              className={cn(
                'flex size-12 items-center justify-center rounded-full backdrop-blur-sm',
                isWide ? 'bg-[rgba(128,255,0,0.25)]' : 'bg-black/50'
              )}
            >
              <svg viewBox='0 0 24 24' fill='white' className='size-5 translate-x-0.5'>
                <path d='M8 5v14l11-7z' />
              </svg>
            </div>
          </div>
        </div>
      </div>

      {modal && <VideoModal src={src} title={title} onClose={() => setModal(false)} />}
    </>
  )
}

export function Gallery() {
  return (
    <section className='px-6 py-10'>
      <div className='mx-auto max-w-[1320px]'>
        {SHOW_CATEGORY_TABS && <div className='mb-8 flex flex-wrap items-center gap-3' />}

        {/* Desktop: 4 explicit columns for precise T/W placement */}
        <div className='grid grid-cols-2 gap-3 lg:grid-cols-4'>
          {GALLERY_COLUMNS.map((col, ci) => (
            <div key={ci} className='flex flex-col gap-3'>
              {col.map((video) => (
                <VideoCard key={video.id} {...video} />
              ))}
            </div>
          ))}
        </div>

        <div className='mt-10 flex justify-center'>
          <button
            className='rounded-full border border-white/10 px-8 py-2 text-base text-[#999] transition-colors hover:border-white/20 hover:text-white'
            style={{ background: 'rgba(255,255,255,0.05)' }}
          >
            查看更多
          </button>
        </div>
      </div>
    </section>
  )
}
