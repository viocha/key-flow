import { useCallback, useEffect, useState } from "react"

export function useDialogTransition(open: boolean, onClose: () => void) {
  const [closing, setClosing] = useState(false)

  useEffect(() => {
    if (!open) setClosing(false)
  }, [open])

  const requestClose = useCallback(() => setClosing(true), [])

  return {
    dialogOpen: open && !closing,
    requestClose,
    onOpenChange: (next: boolean) => {
      if (!next) requestClose()
    },
    onOpenChangeComplete: (next: boolean) => {
      if (!next) onClose()
    },
  }
}
