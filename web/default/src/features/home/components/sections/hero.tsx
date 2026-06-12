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
import { useRef, useState } from 'react'
import { Swiper, SwiperSlide } from 'swiper/react'
import { Autoplay, Pagination } from 'swiper/modules'
import 'swiper/css'
import 'swiper/css/pagination'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

type FeatureItem =
  | { type: 'badge'; iconUrl: string; title: string; badge: string; subtitle: string }
  | { type: 'tags'; iconUrl: string; title: string; tags: string[] }
  | { type: 'simple'; iconUrl: string; title: string; subtitle: string }

const FEATURE_ITEMS: FeatureItem[] = [
  {
    type: 'badge',
    iconUrl: 'https://ai-gc.tos-cn-beijing.volces.com/asset/icon.svg',
    title: '火山引擎 Seedance 2.0',
    badge: '永不溢价',
    subtitle: '官方授权·价格长期稳定',
  },
  {
    type: 'tags',
    iconUrl: 'https://ai-gc.tos-cn-beijing.volces.com/asset/icon-1.svg',
    title: '百万级真人素材库',
    tags: ['人物', '场景', '道具'],
  },
  {
    type: 'simple',
    iconUrl: 'https://ai-gc.tos-cn-beijing.volces.com/asset/icon-2.svg',
    title: '并发端口充裕不排队',
    subtitle: '最高支持 1000 路并发·生成无需等待',
  },
]

const ACTION_BUTTONS = [
  { label: '立即生成视频' },
  { label: '生成图文' },
  { label: '查看素材库' },
  { label: '去任务超市' },
]

function HeroVideoSlide({ src }: { src: string }) {
  const videoRef = useRef<HTMLVideoElement>(null)
  return (
    <div className='relative h-full w-full bg-black'>
      <video
        ref={videoRef}
        src={src}
        autoPlay
        loop
        muted
        playsInline
        className='h-full w-full object-cover'
      />
    </div>
  )
}

export function Hero(_props: HeroProps) {
  const [active, setActive] = useState('立即生成视频')
  return (
    <section
      className='relative px-6 pt-28 pb-8 md:pt-36 md:pb-10'
      style={{
        backgroundImage: 'url(https://ai-gc.tos-cn-beijing.volces.com/asset/light.png)',
        backgroundSize: 'cover',
        backgroundPosition: 'center top',
        backgroundRepeat: 'no-repeat',
      }}
    >
      {/* Dark overlay for readability */}
      <div className='pointer-events-none absolute inset-0 bg-[#0a0a0a]/60' />
      <div className='relative mx-auto max-w-[1320px]'>
        <div className='grid grid-cols-1 items-start gap-12 lg:grid-cols-12'>
          {/* Left: Feature items */}
          <div className='flex flex-col gap-8 lg:col-span-5'>
            {FEATURE_ITEMS.map((item) => (
              <div key={item.title} className='flex flex-col gap-3'>
                {/* Icon + title row */}
                <div className='flex flex-col gap-1'>
                  <div className='flex items-center gap-3'>
                    {/* Icon: 54×54, no background (matches Figma) */}
                    <img src={item.iconUrl} alt='' className='size-[54px] shrink-0' />
                    <span className='text-[36px] font-bold leading-tight text-white'>
                      {item.title}
                    </span>
                  </div>
                  {item.type === 'badge' && (
                    <span className='ml-[66px] text-[36px] font-bold leading-tight text-white'>
                      {item.badge}
                    </span>
                  )}
                </div>
                {/* Subtitle — 20px regular */}
                {(item.type === 'badge' || item.type === 'simple') && (
                  <p className='text-[20px] leading-relaxed text-[#808080]'>
                    {item.subtitle}
                  </p>
                )}
                {/* Tags */}
                {item.type === 'tags' && (
                  <div className='flex flex-wrap items-center gap-3'>
                    {item.tags.map((tag) => (
                      <span
                        key={tag}
                        className='rounded-full px-5 py-1.5 text-base text-[#808080]'
                        style={{
                          background: 'rgba(255,255,255,0.05)',
                          border: '1px solid rgba(255,255,255,0.1)',
                        }}
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* Right: Swiper carousel */}
          <div className='flex flex-col items-center gap-6 lg:col-span-7'>
            <div className='hero-swiper w-full overflow-hidden rounded-3xl' style={{ height: 370 }}>
              <Swiper
                modules={[Autoplay, Pagination]}
                autoplay={{ delay: 5000, disableOnInteraction: false, pauseOnMouseEnter: true }}
                pagination={{ clickable: true, el: '.hero-swiper-pagination' }}
                loop
                className='h-full w-full'
              >
                {/* Slide 1: Video */}
                <SwiperSlide>
                  <HeroVideoSlide src='https://www.heixiu.net/videos/banner-video1.mp4' />
                </SwiperSlide>
                {/* Slide 2: Image */}
                <SwiperSlide>
                  <img
                    src='https://ai-gc.tos-cn-beijing.volces.com/asset/8a577abc5d5a45eec3287819666c1e70.jpg'
                    alt='banner'
                    className='h-full w-full object-cover'
                  />
                </SwiperSlide>
              </Swiper>
            </div>
            {/* Custom pagination dots */}
            <div className='hero-swiper-pagination flex items-center justify-center gap-3' />
          </div>
        </div>

        {/* Action buttons */}
        <div className='mt-14 flex flex-wrap items-center justify-center gap-5'>
          {ACTION_BUTTONS.map((btn) => {
            const isActive = active === btn.label
            return isActive ? (
              <div
                key={btn.label}
                onClick={() => setActive(btn.label)}
                className='cursor-pointer rounded-full p-px'
                style={{ background: 'linear-gradient(180deg, #80FF00 0%, #FBFF00 100%)' }}
              >
                <div className='flex w-[248px] items-center justify-center rounded-full bg-[#171717] px-10 py-[14px] text-lg font-medium text-white'>
                  {btn.label}
                </div>
              </div>
            ) : (
              <div
                key={btn.label}
                onClick={() => setActive(btn.label)}
                className='flex w-[248px] cursor-pointer items-center justify-center rounded-full border border-[#4D4D4D] bg-white/[0.03] px-10 py-[14px] text-lg font-medium text-white transition-all hover:border-white/30 hover:bg-white/[0.07]'
              >
                {btn.label}
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}
