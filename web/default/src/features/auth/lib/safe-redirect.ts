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

export const DEFAULT_REDIRECT = '/dashboard'

export function safeRedirectPath(
  input: unknown,
  fallback: string = DEFAULT_REDIRECT
): string {
  if (typeof input !== 'string' || input.length === 0) return fallback

  // Must start with a single "/" followed by a non-slash, non-backslash char.
  // Rejects "//evil.com", "/\evil.com", protocol-relative and empty roots.
  if (!/^\/[^/\\]/.test(input)) return fallback

  // Reject anything that could resolve to an absolute URL or a JS/data scheme.
  if (/[\s\x00-\x1f]/.test(input)) return fallback
  if (/^\/(?:https?:|javascript:|data:|vbscript:|file:)/i.test(input)) {
    return fallback
  }

  // Final defense: parse against the current origin and confirm same-origin.
  try {
    const origin =
      typeof window !== 'undefined' && window.location?.origin
        ? window.location.origin
        : 'http://localhost'
    const url = new URL(input, origin)
    if (url.origin !== origin) return fallback
    return url.pathname + url.search + url.hash
  } catch {
    return fallback
  }
}
