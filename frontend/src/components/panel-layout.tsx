import type { ReactNode } from "react"

import { ScrollArea } from "@/components/ui/scroll-area"

export function PanelLayout({
  header,
  children,
}: {
  header: ReactNode
  children: ReactNode
}) {
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="mx-auto w-full max-w-5xl shrink-0 px-6 pt-6 pb-4">
        {header}
      </div>
      <ScrollArea className="min-h-0 flex-1">
        <div className="mx-auto flex w-full max-w-5xl flex-col gap-4 px-6 pt-1 pb-6">
          {children}
        </div>
      </ScrollArea>
    </div>
  )
}
