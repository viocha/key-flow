import { toast } from "@/components/ui/toast"

export function showUndoDeleteToast(label: string, undo: () => void) {
  let toastID = ""
  toastID = toast.add({
    title: `已删除${label}`,
    timeout: 5000,
    actionProps: {
      children: "撤销",
      onClick: () => {
        undo()
        toast.close(toastID)
      },
    },
  })
}
