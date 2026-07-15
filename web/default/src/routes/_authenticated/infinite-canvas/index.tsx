import { createFileRoute } from '@tanstack/react-router'
import { Main } from '@/components/layout'

export const Route = createFileRoute('/_authenticated/infinite-canvas/')({
  component: InfiniteCanvasPage,
})

function InfiniteCanvasPage() {
  const baseUrl = window.location.origin
  // 直接落到画布项目列表（/canvas 路由），跳过 infinite-canvas 自带的 HomePage。
  // embedded=1 让画布 SPA 隐藏自有品牌（top-nav logo、h1 标题、页面 title）。
  const src = `/canvas/canvas?baseUrl=${encodeURIComponent(baseUrl)}&embedded=1`

  return (
    <Main className='p-0'>
      <iframe
        src={src}
        title='Infinite Canvas'
        className='h-full w-full border-0'
        allow='clipboard-read; clipboard-write'
      />
    </Main>
  )
}
