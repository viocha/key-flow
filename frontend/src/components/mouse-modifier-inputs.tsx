import {
  ArrowBigUpIcon,
  ChevronUpIcon,
  CommandIcon,
  OptionIcon,
} from "lucide-react"

import { Field, FieldLabel } from "@/components/ui/field"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { WheelStepsInput } from "@/components/wheel-setting-inputs"
import { normalizeMouseModifiers } from "@/lib/app-model"
import { cn } from "@/lib/utils"

const modifiers = [
  { name: "Win", icon: CommandIcon },
  { name: "Ctrl", icon: ChevronUpIcon },
  { name: "Alt", icon: OptionIcon },
  { name: "Shift", icon: ArrowBigUpIcon },
] as const

export function MouseModifierInputs({
  value,
  onChange,
  wheelSteps,
  onWheelStepsChange,
}: {
  value?: string[] | null
  onChange: (modifiers: string[]) => void
  wheelSteps?: number
  onWheelStepsChange?: (steps: number) => void
}) {
  return (
    <div
      className={cn(
        "grid gap-4",
        wheelSteps !== undefined && "grid-cols-[minmax(0,1fr)_auto]",
      )}
    >
      <Field className="min-w-0">
        <FieldLabel>修饰键</FieldLabel>
        <ToggleGroup
          multiple
          variant="outline"
          className="w-full"
          aria-label="修饰键"
          value={normalizeMouseModifiers(value)}
          onValueChange={(selected) =>
            onChange(normalizeMouseModifiers(selected))
          }
        >
          {modifiers.map(({ name, icon: Icon }) => (
            <ToggleGroupItem
              key={name}
              value={name}
              className="min-w-8 flex-1"
              aria-label={name}
              title={name}
            >
              <Icon aria-hidden="true" />
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </Field>
      {wheelSteps !== undefined && onWheelStepsChange && (
        <WheelStepsInput value={wheelSteps} onValueChange={onWheelStepsChange} />
      )}
    </div>
  )
}
