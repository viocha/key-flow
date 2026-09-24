import { useCallback, useEffect, useRef } from "react"
import { AppService } from "../../bindings/key-flow"

export function useInputCapture() {
  const activeRef = useRef(false)

  const begin = useCallback(() => {
    if (activeRef.current) return
    activeRef.current = true
    void AppService.BeginInputCapture()
  }, [])

  const end = useCallback(() => {
    if (!activeRef.current) return
    activeRef.current = false
    void AppService.EndInputCapture()
  }, [])

  useEffect(() => end, [end])

  return { begin, end }
}
