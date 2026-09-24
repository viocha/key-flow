import { useId } from "react"

import { Field, FieldLabel } from "@/components/ui/field"
import { NumberInput } from "@/components/ui/number-input"

export function WheelStepsInput({
  value,
  onValueChange,
}: {
  value: number
  onValueChange: (value: number) => void
}) {
  const inputId = useId()
  return (
    <Field>
      <FieldLabel htmlFor={inputId}>灵敏度</FieldLabel>
      <div className="flex items-center gap-2 whitespace-nowrap text-sm">
        <span>每</span>
        <NumberInput
          id={inputId}
          className="w-16 shrink-0"
          min={1}
          max={20}
          step={1}
          value={value}
          onValueChange={(next) => onValueChange(Math.round(next))}
          aria-label="滚轮触发格数"
        />
        <span>格触发一次</span>
      </div>
    </Field>
  )
}

export function WheelChangeInput({
  value,
  max,
  unit,
  onValueChange,
}: {
  value: number
  max: number
  unit: "%" | "档"
  onValueChange: (value: number) => void
}) {
  return (
    <Field orientation="horizontal" className="w-auto shrink-0">
      <FieldLabel className="whitespace-nowrap font-normal">
        每次变化
        <NumberInput
          className="w-16 shrink-0"
          min={1}
          max={max}
          step={1}
          value={value}
          onValueChange={(next) => onValueChange(Math.round(next))}
          aria-label="每次变化量"
        />
        {unit}
      </FieldLabel>
    </Field>
  )
}
