import { useEffect, useState } from "react"
import type { Action, ActionPreset } from "../../bindings/key-flow"

import { ActionEditor } from "@/components/action-editor"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { NumberInput } from "@/components/ui/number-input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  bidirectionalOperation,
  makeBidirectionalMouseActions,
  type BidirectionalOperation,
} from "@/lib/app-model"

const operationItems: Record<BidirectionalOperation, string> = {
  volume: "音量",
  brightness: "亮度",
  opacity: "窗口透明度",
  scale: "窗口中心缩放",
  switchApps: "切换应用",
  switchTabs: "切换标签页",
  page: "翻页",
  custom: "自定义动作",
}

const amountSettings = {
  volume: { max: 100, initial: 5, unit: "%" },
  brightness: { max: 100, initial: 5, unit: "%" },
  opacity: { max: 100, initial: 5, unit: "%" },
  scale: { max: 100, initial: 5, unit: "%" },
} as const

export function BidirectionalActionEditor({
  input,
  action,
  oppositeAction,
  presets,
  onChange,
}: {
  input: string
  action: Action
  oppositeAction: Action
  presets: ActionPreset[]
  onChange: (action: Action, oppositeAction: Action) => void
}) {
  const [operation, setOperation] = useState<BidirectionalOperation>(() =>
    bidirectionalOperation(action, oppositeAction),
  )
  useEffect(() => {
    setOperation(bidirectionalOperation(action, oppositeAction))
  }, [action, oppositeAction])
  const [firstDirection, secondDirection] =
    input === "wheelHorizontal" ? ["向左", "向右"] : ["向上", "向下"]
  const amount = operation in amountSettings
    ? amountSettings[operation as keyof typeof amountSettings]
    : null

  return (
    <FieldGroup>
      <Field>
        <FieldLabel>双向操作</FieldLabel>
        <Select
          items={operationItems}
          value={operation}
          onValueChange={(value) => {
            if (!value) return
            const selected = value as BidirectionalOperation
            setOperation(selected)
            if (selected === "custom") return
            const initial = selected in amountSettings
              ? amountSettings[selected as keyof typeof amountSettings].initial
              : 1
            const [first, second] = makeBidirectionalMouseActions(selected, initial)
            onChange(first, second)
          }}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              {Object.entries(operationItems).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      </Field>

      {amount && (
        <Field>
          <FieldLabel>变化程度</FieldLabel>
          <div className="flex w-full items-center gap-2">
            <NumberInput
              className="min-w-0 flex-1"
              min={1}
              max={amount.max}
              step={1}
              value={Math.abs(action.delta ?? amount.initial)}
              onValueChange={(value) => {
                const next = Math.max(1, Math.min(amount.max, Math.round(value)))
                let [first, second] = makeBidirectionalMouseActions(
                  operation as Exclude<BidirectionalOperation, "custom">,
                  next,
                )
                if ((action.delta ?? 0) < 0 && operation !== "switchApps") {
                  ;[first, second] = [second, first]
                }
                onChange(first, second)
              }}
            />
            <span className="text-sm text-muted-foreground">{amount.unit}</span>
          </div>
        </Field>
      )}

      {operation === "custom" && (
        <>
          <ActionEditor
            action={action}
            presets={presets}
            label={`${firstDirection}动作`}
            onChange={(next) => onChange(next, oppositeAction)}
          />
          <ActionEditor
            action={oppositeAction}
            presets={presets}
            label={`${secondDirection}动作`}
            onChange={(next) => onChange(action, next)}
          />
        </>
      )}
    </FieldGroup>
  )
}
