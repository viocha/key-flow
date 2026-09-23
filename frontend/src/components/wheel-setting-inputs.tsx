import { Field, FieldLabel } from "@/components/ui/field"
import { NumberInput } from "@/components/ui/number-input"

export function WheelStepsInput({
  value,
  onValueChange,
}: {
  value: number
  onValueChange: (value: number) => void
}) {
  return (
    <Field orientation="horizontal" className="w-auto shrink-0">
      <FieldLabel className="whitespace-nowrap">
        每
        <NumberInput
          className="w-16 shrink-0"
          min={1}
          max={20}
          step={1}
          value={value}
          onValueChange={(next) => onValueChange(Math.round(next))}
          aria-label="滚轮触发格数"
        />
        格触发一次
      </FieldLabel>
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
      <FieldLabel className="whitespace-nowrap">
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
