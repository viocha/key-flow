import {
  ActionType,
  EdgeAdjustmentType,
  type Action,
  type AppConfig,
  type EdgeWheelRule,
  type GestureBinding,
  type GestureGroup,
  type HotkeyRule,
  type LayerBinding,
  type MouseShortcutBinding,
  type Profile,
  type TriggerLayer,
} from "../../bindings/key-flow"

export type UiGestureGroup = Omit<GestureGroup, "bindings"> & {
  bindings: GestureBinding[]
}

export type UiLayer = Omit<
  TriggerLayer,
  "bindings" | "mouseShortcuts"
> & {
  bindings: LayerBinding[]
  mouseShortcuts: MouseShortcutBinding[]
}

export type UiProfile = Omit<
  Profile,
  "hotkeys" | "layers" | "gestures" | "edgeWheels"
> & {
  hotkeys: HotkeyRule[]
  layers: UiLayer[]
  gestures: UiGestureGroup[]
  edgeWheels: EdgeWheelRule[]
}

export type UiConfig = Omit<AppConfig, "profiles"> & {
  profiles: UiProfile[]
}

export function defaultGestureGroup(trigger = "MouseMiddle"): UiGestureGroup {
  return {
    id: newID("gesture-group"),
    enabled: true,
    trigger,
    showTrail: true,
    startDistancePx: 18,
    segmentDistancePx: 28,
    bindings: [],
  }
}

export function normalizeConfig(config: AppConfig): UiConfig {
  return {
    ...config,
    profiles: (config.profiles ?? []).map((profile) => ({
      ...profile,
      hotkeys: profile.hotkeys ?? [],
      gestures: (profile.gestures ?? []).map((group) => ({
        ...group,
        bindings: group.bindings ?? [],
      })),
      edgeWheels: profile.edgeWheels ?? [],
      layers: (profile.layers ?? []).map((layer) => ({
        ...layer,
        bindings: layer.bindings ?? [],
        mouseShortcuts: layer.mouseShortcuts ?? [],
        windowOperations: {
          enabled: layer.windowOperations?.enabled ?? false,
          moveButton: layer.windowOperations?.moveButton ?? "",
          resizeButton: layer.windowOperations?.resizeButton ?? "",
        },
      })),
    })),
  }
}

export function cloneAction(action: Action): Action {
  return JSON.parse(JSON.stringify(action)) as Action
}

export function newID(prefix: string) {
  const suffix =
    typeof crypto !== "undefined" && "randomUUID" in crypto
      ? crypto.randomUUID()
      : Date.now() + "-" + Math.random().toString(16).slice(2)
  return prefix + "-" + suffix
}

export function actionSummary(action: Action) {
  switch (action.type) {
    case ActionType.ActionKeyStroke:
      return orderShortcutKeys(action.keys).join(" + ")
    case ActionType.ActionText:
      return action.text ? "输入 “" + shortText(action.text) + "”" : "输入文本"
    case ActionType.ActionLaunch:
      return action.target ? "打开 " + action.target : "打开目标"
    case ActionType.ActionCommand:
      return action.program ? "运行 " + action.program : "运行命令"
    case ActionType.ActionSequence:
      return "动作序列 · " + (action.sequence?.length ?? 0) + " 项"
    case ActionType.ActionEdit:
      return editCommandLabel(action.editCommand ?? "")
    case ActionType.ActionWindow:
      if (action.windowCommand === "opacity") {
        const delta = action.delta ?? 0
        return delta > 0
          ? `透明度 +${delta}%`
          : `透明度 ${delta}%`
      }
      if (action.windowCommand === "scale") {
        const delta = action.delta ?? 0
        return delta > 0
          ? `窗口缩放 +${delta}%`
          : `窗口缩放 ${delta}%`
      }
      return windowCommandLabel(action.windowCommand ?? "")
    case ActionType.ActionMouse:
      return mouseCommandLabel(action.mouseCommand ?? "") +
        (action.mouseCommand?.startsWith("wheel") && (action.delta ?? 1) > 1
          ? ` ${action.delta} 格`
          : "")
    case ActionType.ActionVolume:
      return action.delta && action.delta > 0
        ? "音量 +" + action.delta + "%"
        : "音量 " + (action.delta ?? 0) + "%"
    case ActionType.ActionBrightness:
      return action.delta && action.delta > 0
        ? "亮度 +" + action.delta + "%"
        : "亮度 " + (action.delta ?? 0) + "%"
    default:
      return "未配置动作"
  }
}

export function editCommandLabel(command: string) {
  switch (command) {
    case "deleteLine":
      return "删除当前行"
    case "insertLineBelow":
      return "下方插入空行"
    case "insertLineAbove":
      return "上方插入空行"
    case "selectLine":
      return "选中当前行"
    case "selectLineWithBreak":
      return "选中当前行（含换行）"
    default:
      return "编辑操作"
  }
}

export function gestureBindingLabel(binding: GestureBinding) {
  const customName = binding.name?.trim()
  if (customName) return customName
  if (
    binding.action.type === ActionType.ActionWindow ||
    binding.action.type === ActionType.ActionVolume ||
    binding.action.type === ActionType.ActionBrightness ||
    (binding.action.type === ActionType.ActionMouse &&
      binding.action.mouseCommand?.startsWith("wheel"))
  ) {
    return actionSummary(binding.action)
  }
  const actionName = binding.action.name?.trim()
  return actionName || actionSummary(binding.action)
}

export function windowCommandLabel(command: string) {
  switch (command) {
    case "close":
      return "关闭窗口"
    case "terminateProcess":
      return "终止进程"
    case "previousWindow":
      return "切换上一个窗口"
    case "nextWindow":
      return "切换下一个窗口"
    case "minimize":
      return "最小化窗口"
    case "restoreMinimized":
      return "恢复最小化窗口"
    case "maximize":
      return "最大化窗口"
    case "restore":
      return "恢复窗口"
    case "toggleMaximize":
      return "切换最大化"
    case "toggleVerticalMaximize":
      return "切换垂直最大化"
    case "opacity":
      return "更改窗口透明度"
    case "scale":
      return "窗口中心缩放"
    case "alwaysOnTop":
      return "切换置顶"
    default:
      return "窗口操作"
  }
}

export function mouseCommandLabel(command: string) {
  const labels: Record<string, string> = {
    leftClick: "左键单击",
    doubleClick: "左键双击",
    rightClick: "右键单击",
    middleClick: "中键单击",
    backClick: "后退键单击",
    forwardClick: "前进键单击",
    wheelUp: "向上滚动",
    wheelDown: "向下滚动",
    wheelLeft: "向左滚动",
    wheelRight: "向右滚动",
  }
  return labels[command] ?? "鼠标操作"
}

export function mouseInputLabel(input: string) {
  switch (input) {
    case "left":
      return "左键"
    case "right":
      return "右键"
    case "middle":
      return "中键"
    case "back":
      return "后退键"
    case "forward":
      return "前进键"
    case "wheelUp":
      return "滚轮向上"
    case "wheelDown":
      return "滚轮向下"
    case "wheelLeft":
      return "滚轮向左"
    case "wheelRight":
      return "滚轮向右"
    case "wheelVertical":
      return "垂直滚动"
    case "wheelHorizontal":
      return "水平滚动"
    default:
      return input
  }
}

const mouseModifierOrder = ["Win", "Ctrl", "Alt", "Shift"] as const

export function orderShortcutKeys(keys?: readonly string[] | null) {
  const ordered = normalizeMouseModifiers(keys)
  const mainKeys = (keys ?? []).filter(
    (key) => !mouseModifierOrder.includes(key as (typeof mouseModifierOrder)[number]),
  )
  return [...ordered, ...mainKeys]
}

export function normalizeMouseModifiers(modifiers?: readonly string[] | null) {
  const selected = new Set(modifiers ?? [])
  return mouseModifierOrder.filter((modifier) => selected.has(modifier))
}

export function mouseShortcutKey(binding: MouseShortcutBinding) {
  return [...normalizeMouseModifiers(binding.modifiers), binding.input].join("+")
}

export function mouseShortcutInputs(input: string) {
  if (input === "wheelVertical") return ["wheelUp", "wheelDown"]
  if (input === "wheelHorizontal") return ["wheelLeft", "wheelRight"]
  return [input]
}

export function mouseShortcutKeys(binding: Pick<MouseShortcutBinding, "input" | "modifiers">) {
  return mouseShortcutInputs(binding.input).map((input) =>
    [...normalizeMouseModifiers(binding.modifiers), input].join("+"),
  )
}

export function hotkeyShortcutKeys(
  rule: Pick<HotkeyRule, "keys" | "mouseInput" | "modifiers">,
) {
  if (rule.mouseInput) {
    return mouseShortcutKeys({
      input: rule.mouseInput,
      modifiers: rule.modifiers,
    }).map((key) => "mouse:" + key)
  }
  if (!rule.keys?.length) return []
  return ["key:" + orderShortcutKeys(rule.keys).join("+")]
}

export function hotkeyConflicts(rule: HotkeyRule, existingRules: HotkeyRule[]) {
  if (!rule.enabled) return false
  const keys = new Set(hotkeyShortcutKeys(rule))
  if (keys.size === 0) return false
  return existingRules.some(
    (existing) =>
      existing.id !== rule.id &&
      existing.enabled &&
      hotkeyShortcutKeys(existing).some((key) => keys.has(key)),
  )
}

export function layerBindingConflicts(
  binding: LayerBinding,
  existingBindings: LayerBinding[],
) {
  if (!binding.keys?.length) return false
  const chord = orderShortcutKeys(binding.keys).join("+")
  return existingBindings.some(
    (existing) =>
      existing.id !== binding.id &&
      orderShortcutKeys(existing.keys).join("+") === chord,
  )
}

export function isBidirectionalMouseInput(input: string) {
  return input === "wheelVertical" || input === "wheelHorizontal"
}

export type BidirectionalOperation =
  | "volume"
  | "brightness"
  | "opacity"
  | "scale"
  | "switchApps"
  | "switchTabs"
  | "page"
  | "custom"

function matchesKeyStrokePair(
  action: Action,
  oppositeAction: Action,
  first: string[],
  second: string[],
) {
  return action.type === ActionType.ActionKeyStroke &&
    oppositeAction.type === ActionType.ActionKeyStroke &&
    orderShortcutKeys(action.keys).join("+") === orderShortcutKeys(first).join("+") &&
    orderShortcutKeys(oppositeAction.keys).join("+") === orderShortcutKeys(second).join("+")
}

export function bidirectionalOperation(action: Action, oppositeAction: Action): BidirectionalOperation {
  const amount = action.delta ?? 0
  const oppositeAmount = oppositeAction.delta ?? 0
  if (amount !== 0 && oppositeAmount === -amount && action.type === oppositeAction.type) {
    if (action.type === ActionType.ActionVolume) return "volume"
    if (action.type === ActionType.ActionBrightness) return "brightness"
    if (
      action.type === ActionType.ActionWindow &&
      action.windowCommand === "opacity" &&
      oppositeAction.windowCommand === "opacity"
    ) return "opacity"
    if (
      action.type === ActionType.ActionWindow &&
      action.windowCommand === "scale" &&
      oppositeAction.windowCommand === "scale"
    ) return "scale"
  }
  if (
    action.type === ActionType.ActionWindow &&
    oppositeAction.type === ActionType.ActionWindow &&
    action.windowCommand === "previousWindow" &&
    oppositeAction.windowCommand === "nextWindow"
  ) return "switchApps"
  if (matchesKeyStrokePair(action, oppositeAction, ["Ctrl", "Shift", "Tab"], ["Ctrl", "Tab"])) {
    return "switchTabs"
  }
  if (matchesKeyStrokePair(action, oppositeAction, ["PageUp"], ["PageDown"])) {
    return "page"
  }
  return "custom"
}

export function makeBidirectionalMouseActions(
  operation: Exclude<BidirectionalOperation, "custom">,
  amount: number,
): [Action, Action] {
  switch (operation) {
    case "brightness":
      return [
        { type: ActionType.ActionBrightness, name: "亮度增加", delta: amount },
        { type: ActionType.ActionBrightness, name: "亮度降低", delta: -amount },
      ]
    case "opacity":
      return [
        { type: ActionType.ActionWindow, name: "增加透明度", windowCommand: "opacity", delta: amount },
        { type: ActionType.ActionWindow, name: "降低透明度", windowCommand: "opacity", delta: -amount },
      ]
    case "scale":
      return [
        { type: ActionType.ActionWindow, name: "放大窗口", windowCommand: "scale", delta: amount },
        { type: ActionType.ActionWindow, name: "缩小窗口", windowCommand: "scale", delta: -amount },
      ]
    case "switchApps":
      return [
        { type: ActionType.ActionWindow, name: "切换上一个窗口", windowCommand: "previousWindow" },
        { type: ActionType.ActionWindow, name: "切换下一个窗口", windowCommand: "nextWindow" },
      ]
    case "switchTabs":
      return [
        { type: ActionType.ActionKeyStroke, name: "上一个标签页", keys: ["Ctrl", "Shift", "Tab"] },
        { type: ActionType.ActionKeyStroke, name: "下一个标签页", keys: ["Ctrl", "Tab"] },
      ]
    case "page":
      return [
        { type: ActionType.ActionKeyStroke, name: "向上翻页", keys: ["PageUp"] },
        { type: ActionType.ActionKeyStroke, name: "向下翻页", keys: ["PageDown"] },
      ]
    default:
      return [
        { type: ActionType.ActionVolume, name: "音量增加", delta: amount },
        { type: ActionType.ActionVolume, name: "音量降低", delta: -amount },
      ]
  }
}

export function defaultBidirectionalMouseActions(_input: string): [Action, Action] {
  return makeBidirectionalMouseActions("volume", 5)
}

export function actionsForMouseInputChange(
  previousInput: string,
  input: string,
  action: Action,
  oppositeAction?: Action | null,
): [Action, Action | undefined] {
  const wasBidirectional = isBidirectionalMouseInput(previousInput)
  if (isBidirectionalMouseInput(input)) {
    if (wasBidirectional && oppositeAction) return [action, oppositeAction]
    const [first, second] = defaultBidirectionalMouseActions(input)
    const directions = mouseShortcutInputs(input)
    if (previousInput === directions[0]) return [action, second]
    if (previousInput === directions[1]) return [first, action]
    return [first, second]
  }
  if (wasBidirectional && input === mouseShortcutInputs(previousInput)[1]) {
    return [oppositeAction ?? action, undefined]
  }
  return [action, undefined]
}

export function mouseShortcutActionSummary(
  input: string,
  action: Action,
  oppositeAction?: Action | null,
) {
  if (!isBidirectionalMouseInput(input) || !oppositeAction) {
    return actionSummary(action)
  }
  const operation = bidirectionalOperation(action, oppositeAction)
  const amount = Math.abs(action.delta ?? 0)
  if (operation === "volume") return `音量 ±${amount}%`
  if (operation === "brightness") return `亮度 ±${amount}%`
  if (operation === "opacity") return `透明度 ±${amount}%`
  if (operation === "scale") return `窗口缩放 ±${amount}%`
  if (operation === "switchApps") return "切换应用"
  if (operation === "switchTabs") return "切换标签页"
  if (operation === "page") return "翻页"
  const [first, second] = input === "wheelHorizontal" ? ["←", "→"] : ["↑", "↓"]
  return `${first} ${actionSummary(action)}  ·  ${second} ${actionSummary(oppositeAction)}`
}

export function mouseShortcutLabel(binding: MouseShortcutBinding) {
  return [
    ...normalizeMouseModifiers(binding.modifiers),
    mouseInputLabel(binding.input),
  ].join(" + ")
}

export function hotkeyInputKeys(rule: HotkeyRule) {
  if (rule.mouseInput) {
    return [
      ...normalizeMouseModifiers(rule.modifiers),
      mouseInputLabel(rule.mouseInput),
    ]
  }
  return rule.keys ?? []
}

export function mouseInputForTrigger(trigger: string) {
  switch (trigger) {
    case "MouseRight":
      return "right"
    case "MouseMiddle":
      return "middle"
    case "MouseBack":
      return "back"
    case "MouseForward":
      return "forward"
    default:
      return ""
  }
}

export function triggerLabel(trigger: string) {
  switch (trigger) {
    case "CapsLock":
      return "CapsLock"
    case "MouseRight":
      return "鼠标右键"
    case "MouseMiddle":
      return "鼠标中键"
    case "MouseBack":
      return "鼠标后退键"
    case "MouseForward":
      return "鼠标前进键"
    default:
      return trigger
  }
}

export function edgeAdjustmentLabel(adjustment: EdgeAdjustmentType) {
  switch (adjustment) {
    case EdgeAdjustmentType.EdgeAdjustmentVolume:
      return "音量"
    case EdgeAdjustmentType.EdgeAdjustmentBrightness:
      return "亮度"
    case EdgeAdjustmentType.EdgeAdjustmentRecentApps:
      return "浏览最近应用"
    case EdgeAdjustmentType.EdgeAdjustmentSwitchApps:
      return "切换应用"
    case EdgeAdjustmentType.EdgeAdjustmentSwitchTabs:
      return "切换标签页"
    case EdgeAdjustmentType.EdgeAdjustmentPage:
      return "翻页"
    case EdgeAdjustmentType.EdgeAdjustmentCustom:
      return "自定义动作"
    default:
      return adjustment
  }
}

export function edgeWheelInputLabel(input: string) {
  return input === "wheelHorizontal" ? "水平滚动" : "垂直滚动"
}

export function edgeWheelConflicts(rule: EdgeWheelRule, existingRules: EdgeWheelRule[]) {
  if (!rule.enabled || !rule.zone || !rule.input) return false
  return existingRules.some(
    (existing) =>
      existing.id !== rule.id &&
      existing.enabled &&
      existing.zone === rule.zone &&
      existing.input === rule.input,
  )
}

export function edgeStepLabel(rule: EdgeWheelRule) {
  return rule.adjustment === EdgeAdjustmentType.EdgeAdjustmentBrightness ||
    rule.adjustment === EdgeAdjustmentType.EdgeAdjustmentVolume
    ? rule.step + "%"
    : rule.step + " 步"
}

export function edgeZoneLabel(zone: string) {
  switch (zone) {
    case "topLeft":
      return "左上角"
    case "top":
      return "上边缘"
    case "topRight":
      return "右上角"
    case "left":
      return "左边缘"
    case "right":
      return "右边缘"
    case "bottomLeft":
      return "左下角"
    case "bottom":
      return "下边缘"
    case "bottomRight":
      return "右下角"
    default:
      return zone
  }
}

export function gestureDirectionLabel(directions: string[] | null | undefined) {
  const map: Record<string, string> = {
    U: "↑",
    D: "↓",
    L: "←",
    R: "→",
    UL: "↖",
    UR: "↗",
    DL: "↙",
    DR: "↘",
  }
  return (directions ?? []).map((direction) => map[direction] ?? direction).join(" ")
}

function gestureBindingKey(directions?: readonly string[] | null) {
  return (directions ?? [])
    .map((direction) => direction.trim().toUpperCase())
    .filter(Boolean)
    .filter((direction, index, values) => index === 0 || direction !== values[index - 1])
    .join(",")
}

export function gestureBindingConflicts(
  binding: GestureBinding,
  existingBindings: GestureBinding[],
) {
  if (!binding.enabled || !binding.directions?.length) return false
  const key = gestureBindingKey(binding.directions)
  return existingBindings.some(
    (existing) =>
      existing.id !== binding.id &&
      existing.enabled &&
      gestureBindingKey(existing.directions) === key,
  )
}

export function defaultAction(): Action {
  return {
    type: ActionType.ActionKeyStroke,
    name: "复制",
    keys: ["Ctrl", "C"],
  }
}

export function emptyAction(): Action {
  return {
    type: ActionType.ActionKeyStroke,
    keys: [],
  }
}

export function actionOrDefault(action: Action | null | undefined): Action {
  return action?.type ? action : defaultAction()
}

export function defaultHotkey(): HotkeyRule {
  return {
    id: newID("hotkey"),
    enabled: true,
    keys: [],
    action: defaultAction(),
  }
}

export function defaultLayer(holdMs: number): UiLayer {
  return {
    id: newID("layer"),
    name: "CapsLock",
    enabled: true,
    trigger: "CapsLock",
    holdMs,
    tapBehavior: "original",
    bindings: [],
    mouseShortcuts: [],
    windowOperations: {
      enabled: false,
      moveButton: "",
      resizeButton: "",
    },
  }
}

export function defaultBinding(): LayerBinding {
  return {
    id: newID("binding"),
    enabled: true,
    keys: [],
    action: {
      type: ActionType.ActionKeyStroke,
      name: "向左",
      keys: ["ArrowLeft"],
    },
  }
}

export function defaultGesture(): GestureBinding {
  return {
    id: newID("gesture"),
    name: "",
    enabled: true,
    directions: [],
    preview: [],
    action: {
      type: ActionType.ActionWindow,
      name: "最小化窗口",
      windowCommand: "minimize",
    },
  }
}

export function defaultMouseShortcutAction(): Action {
  return {
    type: ActionType.ActionWindow,
    name: "切换最大化",
    windowCommand: "toggleMaximize",
  }
}

export function defaultEdgeWheel(): EdgeWheelRule {
  return {
    id: newID("edge"),
    enabled: true,
    zone: "",
    input: "",
    adjustment: EdgeAdjustmentType.EdgeAdjustmentVolume,
    step: 5,
    wheelSteps: 1,
  }
}

export function activeProfile(config: UiConfig): UiProfile | undefined {
  return config.profiles.find((profile) => profile.id === config.activeProfileId)
}

export function shortText(value: string, length = 32) {
  const compact = value.replace(/s+/g, " ").trim()
  return compact.length > length ? compact.slice(0, length) + "…" : compact
}
