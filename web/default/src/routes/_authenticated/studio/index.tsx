import { createFileRoute } from '@tanstack/react-router'
import { AppHeader, Main } from '@/components/layout'
import { Studio } from '@/features/studio'

export const Route = createFileRoute('/_authenticated/studio/')({
  component: StudioPage,
})

function StudioPage() {
  return (
    <>
      <AppHeader />
      <Main className='overflow-y-auto'>
        <Studio />
      </Main>
    </>
  )
}
