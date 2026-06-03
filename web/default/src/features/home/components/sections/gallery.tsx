import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { AnimateInView } from '@/components/animate-in-view'

const CATEGORIES = ['All', 'Chat', 'Image', 'Video', 'Audio', 'Embedding'] as const

interface ShowcaseCard {
  id: number
  provider: string
  model: string
  category: (typeof CATEGORIES)[number]
  desc: string
  accent: string
  height: number
}

const SHOWCASE_CARDS: ShowcaseCard[] = [
  { id: 1, provider: 'OpenAI', model: 'GPT-4o', category: 'Chat', desc: 'Advanced multimodal model for text and images', accent: '#74aa9c', height: 180 },
  { id: 2, provider: 'Anthropic', model: 'Claude 3.5 Sonnet', category: 'Chat', desc: 'Powerful assistant with 200K context window', accent: '#c96442', height: 220 },
  { id: 3, provider: 'Google', model: 'Gemini 2.5 Pro', category: 'Chat', desc: 'Multimodal AI with native long context', accent: '#4285f4', height: 160 },
  { id: 4, provider: 'DeepSeek', model: 'DeepSeek-V3', category: 'Chat', desc: 'High-performance open-source model', accent: '#3d6aff', height: 200 },
  { id: 5, provider: 'Stability AI', model: 'Stable Diffusion XL', category: 'Image', desc: 'State-of-the-art image generation', accent: '#9b59b6', height: 240 },
  { id: 6, provider: 'OpenAI', model: 'DALL-E 3', category: 'Image', desc: 'Create realistic images from natural language', accent: '#1d9bf0', height: 180 },
  { id: 7, provider: 'ElevenLabs', model: 'Eleven v2', category: 'Audio', desc: 'Natural voice synthesis and voice cloning', accent: '#2ecc71', height: 200 },
  { id: 8, provider: 'OpenAI', model: 'Whisper Large v3', category: 'Audio', desc: 'Industry-leading speech recognition', accent: '#e67e22', height: 160 },
  { id: 9, provider: 'OpenAI', model: 'Sora', category: 'Video', desc: 'World-class text-to-video generation', accent: '#e74c3c', height: 220 },
  { id: 10, provider: 'OpenAI', model: 'text-embedding-3', category: 'Embedding', desc: 'Semantic text vector embeddings', accent: '#74aa9c', height: 180 },
  { id: 11, provider: 'Alibaba', model: 'Qwen2.5-Max', category: 'Chat', desc: 'Alibaba latest multimodal model', accent: '#ff6a00', height: 200 },
  { id: 12, provider: 'Meta', model: 'Llama 3.1 405B', category: 'Chat', desc: 'Meta open-source flagship model', accent: '#0064e0', height: 160 },
  { id: 13, provider: 'Midjourney', model: 'Midjourney v6', category: 'Image', desc: 'Artistic AI image generation', accent: '#f1c40f', height: 240 },
  { id: 14, provider: 'Runway', model: 'Gen-3 Alpha', category: 'Video', desc: 'High-quality video generation', accent: '#8e44ad', height: 200 },
  { id: 15, provider: 'Cohere', model: 'Embed v3', category: 'Embedding', desc: 'Multilingual embeddings for search', accent: '#16a085', height: 160 },
  { id: 16, provider: 'Mistral AI', model: 'Mistral Large', category: 'Chat', desc: 'European frontier AI model', accent: '#f39c12', height: 180 },
]

export function Gallery() {
  const { t } = useTranslation()
  const [activeCategory, setActiveCategory] = useState<(typeof CATEGORIES)[number]>('All')

  const filtered =
    activeCategory === 'All'
      ? SHOWCASE_CARDS
      : SHOWCASE_CARDS.filter((c) => c.category === activeCategory)

  return (
    <section className='bg-[#171717] px-6 py-20'>
      <div className='mx-auto max-w-7xl'>
        <AnimateInView animation='fade-up' className='mb-10'>
          <p className='mb-2 text-xs font-medium tracking-widest text-[#808080] uppercase'>
            {t('Supported Models')}
          </p>
          <h2 className='text-2xl font-bold text-white md:text-3xl'>
            {t('Explore AI Capabilities')}
          </h2>
        </AnimateInView>

        {/* Category filters */}
        <AnimateInView animation='fade-up' delay={80}>
          <div className='mb-8 flex flex-wrap items-center gap-3'>
            {CATEGORIES.map((c) => (
              <button
                key={c}
                onClick={() => setActiveCategory(c)}
                className={cn(
                  'rounded-full px-5 py-2 text-sm transition-all duration-200',
                  activeCategory === c
                    ? 'border border-white/40 bg-white/10 text-white'
                    : 'border border-[#333] bg-transparent text-[#999] hover:border-white/20 hover:text-white'
                )}
              >
                {t(c)}
              </button>
            ))}
          </div>
        </AnimateInView>

        {/* Masonry grid */}
        <AnimateInView animation='fade-up' delay={160}>
          <div className='columns-2 gap-4 md:columns-3 lg:columns-4'>
            {filtered.map((card) => (
              <div
                key={card.id}
                className='mb-4 break-inside-avoid overflow-hidden rounded-2xl border border-white/[0.06] bg-[#1e1e1e] transition-all duration-300 hover:border-white/20 hover:shadow-lg'
              >
                {/* Card image placeholder */}
                <div
                  className='relative w-full overflow-hidden'
                  style={{
                    height: `${card.height}px`,
                    background: `linear-gradient(135deg, ${card.accent}22 0%, ${card.accent}0a 100%)`,
                  }}
                >
                  {/* Decorative glow circle */}
                  <div
                    className='absolute inset-0 flex items-center justify-center'
                    aria-hidden
                  >
                    <div
                      className='size-16 rounded-full blur-2xl opacity-40'
                      style={{ background: card.accent }}
                    />
                  </div>
                  {/* Provider initial */}
                  <div className='absolute inset-0 flex items-center justify-center'>
                    <span
                      className='text-5xl font-black'
                      style={{ color: `${card.accent}99` }}
                    >
                      {card.provider[0]}
                    </span>
                  </div>
                  {/* Category badge */}
                  <div className='absolute top-3 right-3'>
                    <span className='rounded-full border border-white/10 bg-black/40 px-2 py-0.5 text-[10px] text-white/60 backdrop-blur-sm'>
                      {t(card.category)}
                    </span>
                  </div>
                </div>

                {/* Card info */}
                <div className='p-3'>
                  <div className='mb-1 flex items-center gap-2'>
                    <div
                      className='size-4 shrink-0 rounded-full'
                      style={{ background: card.accent }}
                    />
                    <span className='text-sm font-medium text-white truncate'>{card.provider}</span>
                  </div>
                  <p className='text-xs font-semibold text-white/70'>{card.model}</p>
                  <p className='mt-1 text-xs text-[#666] line-clamp-2'>{t(card.desc)}</p>
                </div>
              </div>
            ))}
          </div>
        </AnimateInView>

        {/* View more button */}
        <AnimateInView animation='fade-up' delay={240} className='mt-8 flex justify-center'>
          <button className='rounded-full border border-white/10 bg-white/[0.05] px-8 py-2.5 text-sm text-white/70 transition-colors hover:bg-white/[0.08] hover:text-white'>
            {t('View More')}
          </button>
        </AnimateInView>
      </div>
    </section>
  )
}
