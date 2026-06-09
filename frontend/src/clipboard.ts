// Copy text to the clipboard, returning whether it succeeded.
//
// navigator.clipboard only works in a "secure context" (https or localhost).
// When the app is opened over plain http on a LAN IP it is undefined/blocked,
// which is why "Copy all" silently did nothing. Fall back to a hidden textarea
// + execCommand('copy'), which works in non-secure contexts too.
export async function copyText(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    /* fall through to the legacy path */
  }
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.top = '-1000px'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}
