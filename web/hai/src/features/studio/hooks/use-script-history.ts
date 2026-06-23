import { useState, useCallback } from 'react'

const STORAGE_KEY = 'studio_script_history'
const MAX_HISTORY = 50

export interface ScriptHistoryItem {
  id: string
  prompt: string
  output: string
  model: string
  timestamp: number
}

function loadHistory(): ScriptHistoryItem[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    return JSON.parse(raw) as ScriptHistoryItem[]
  } catch {
    return []
  }
}

function saveHistory(items: ScriptHistoryItem[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(items))
  } catch {
    // ignore quota errors
  }
}

export function useScriptHistory() {
  const [history, setHistory] = useState<ScriptHistoryItem[]>(() =>
    loadHistory()
  )

  const addRecord = useCallback((item: Omit<ScriptHistoryItem, 'id' | 'timestamp'>) => {
    const newItem: ScriptHistoryItem = {
      ...item,
      id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      timestamp: Date.now(),
    }
    setHistory((prev) => {
      const next = [newItem, ...prev].slice(0, MAX_HISTORY)
      saveHistory(next)
      return next
    })
    return newItem.id
  }, [])

  const deleteRecord = useCallback((id: string) => {
    setHistory((prev) => {
      const next = prev.filter((item) => item.id !== id)
      saveHistory(next)
      return next
    })
  }, [])

  const clearAll = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY)
    setHistory([])
  }, [])

  return { history, addRecord, deleteRecord, clearAll }
}
