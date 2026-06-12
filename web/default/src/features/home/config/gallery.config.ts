/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

export interface GalleryVideo {
  id: number
  /** Display title shown on the card */
  title: string
  /** Video source URL */
  src: string
  /** Optional poster image URL (shown before video loads) */
  poster?: string
  /**
   * Layout size in the grid:
   * - 'wide'  = col-span-2, landscape 16:9
   * - 'tall'  = col-span-1, portrait  9:16
   * - 'normal'= col-span-1, landscape 16:9  (default)
   */
  size?: 'wide' | 'tall' | 'normal'
}

/**
 * Homepage gallery video list.
 * Edit this file to add/remove/reorder videos.
 * Tab category filters are currently hidden — set SHOW_CATEGORY_TABS = true to re-enable.
 */
export const SHOW_CATEGORY_TABS = false

/**
 * 4-column explicit layout:
 * col1: T+T   col2: W+W+T   col3: T+T   col4: T+W+T
 */
export const GALLERY_COLUMNS: GalleryVideo[][] = [
  // col1: tall + tall
  [
    { id: 1, title: '玄幻风格', src: 'https://eclectic-mermaid-cba15d.netlify.app/videos/style_xuanhuan.mp4', size: 'tall' },
    { id: 2, title: '穿越风格', src: 'https://eclectic-mermaid-cba15d.netlify.app/videos/style_chuanyue.mp4', size: 'tall' },
  ],
  // col2: wide + wide + tall
  [
    { id: 3, title: '美漫风格',  src: 'https://eclectic-mermaid-cba15d.netlify.app/videos/style_meiman.mp4',   size: 'wide' },
    { id: 4, title: '校园风格',  src: 'https://ai-gc.tos-cn-beijing.volces.com/asset/59afd7ddd8d3c95059bd1df695061bf7.mp4', size: 'wide' },
    { id: 5, title: '赛博风格',  src: 'https://eclectic-mermaid-cba15d.netlify.app/videos/style_saibo.mp4',   size: 'tall' },
  ],
  // col3: tall + tall
  [
    { id: 6, title: '甜心风格', src: 'https://eclectic-mermaid-cba15d.netlify.app/videos/style_tianxin.mp4', size: 'tall' },
    { id: 7, title: '奇幻风格', src: 'https://eclectic-mermaid-cba15d.netlify.app/videos/style_qihuan.mp4',  size: 'tall' },
  ],
  // col4: tall + wide + tall
  [
    { id: 8,  title: '都市风格', src: 'https://eclectic-mermaid-cba15d.netlify.app/videos/style_dushi.mp4',   size: 'tall' },
    { id: 9,  title: '恐怖风格', src: 'https://eclectic-mermaid-cba15d.netlify.app/videos/style_kongbu.mp4',  size: 'wide' },
    { id: 10, title: '出海精选', src: 'https://ai-gc.tos-cn-beijing.volces.com/asset/ea1a1cec6100e54d86703d8fff71e777.mp4', size: 'tall' },
  ],
]

/** Flat list — kept for any code that needs a simple array */
export const GALLERY_VIDEOS: GalleryVideo[] = GALLERY_COLUMNS.flat()
