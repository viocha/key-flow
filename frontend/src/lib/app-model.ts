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
          scaleWheel: layer.windowOperations?.scaleWheel ?? false,
          scaleStepPercent: layer.windowOperations?.scaleStepPercent || 5,
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
      return (action.keys ?? []).join(" + ")
    case ActionType.ActionText:
      return action.text ? "输入 “" + shortText(action.text) + "”" : "输入文本"
    case ActionType.ActionLaunch:
      return action.target ? "打开 " + action.target : "打开目标"
    case ActionType.ActionCommand:
      return action.program ? "运行 " + action.program : "运行命令"
    case ActionType.ActionSequence:
      return "动作序列 · " + (action.sequence?.length ?? 0) + " 项"
    case ActionType.ActionWindow:
      return windowCommandLabel(action.windowCommand ?? "")
    case ActionType.ActionVolume:
      return action.delta && action.delta > 0
        ? "音量 +" + action.delta
        : "音量 " + (action.delta ?? 0)
    case ActionType.ActionBrightness:
      return action.delta && action.delta > 0
        ? "亮度 +" + action.delta + "%"
        : "亮度 " + (action.delta ?? 0) + "%"
    default:
      return "未配置动作"
  }
}

export function gestureBindingLabel(binding: GestureBinding) {
  const customName = binding.name?.trim()
  if (customName) return customName
  if (binding.action.type === ActionType.ActionWindow) {
    return windowCommandLabel(binding.action.windowCommand ?? "")
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
    case "alwaysOnTop":
      return "切换置顶"
    default:
      return "窗口操作"
  }
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
    default:
      return input
  }
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
      return "最近应用"
    case EdgeAdjustmentType.EdgeAdjustmentBidirectionalKeys:
      return "双向快捷键"
    default:
      return adjustment
  }
}

export function edgeStepLabel(rule: EdgeWheelRule) {
  return rule.adjustment === EdgeAdjustmentType.EdgeAdjustmentBrightness
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

export function defaultAction(): Action {
  return {
    type: ActionType.ActionKeyStroke,
    name: "复制",
    keys: ["Ctrl", "C"],
  }
}

export function actionOrDefault(action: Action | null | undefined): Action {
  return action?.type ? action : defaultAction()
}

export function defaultHotkey(): HotkeyRule {
  return {
    id: newID("hotkey"),
    name: "新快捷键",
    enabled: true,
    keys: ["Ctrl", "Alt", "K"],
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
      scaleWheel: false,
      scaleStepPercent: 5,
    },
  }
}

export function defaultBinding(): LayerBinding {
  return {
    id: newID("binding"),
    keys: ["S"],
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
    zone: "bottomRight",
    adjustment: EdgeAdjustmentType.EdgeAdjustmentVolume,
    step: 1,
    upKeys: [],
    downKeys: [],
  }
}

export function activeProfile(config: UiConfig): UiProfile | undefined {
  return config.profiles.find((profile) => profile.id === config.activeProfileId)
}

export function shortText(value: string, length = 32) {
  const compact = value.replace(/s+/g, " ").trim()
  return compact.length > length ? compact.slice(0, length) + "…" : compact
}
