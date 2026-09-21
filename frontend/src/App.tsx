import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { Events } from "@wailsio/runtime"
import {
  AppService,
  type ActionPreset,
  type HotkeyRule,
  type LayerBinding,
} from "../bindings/key-flow"
import {
  FolderOpenIcon,
  Loader2Icon,
  MoreHorizontalIcon,
  PencilIcon,
  PlusIcon,
  SettingsIcon,
  Trash2Icon,
} from "lucide-react"

import { EdgeWheelPanel } from "@/components/edge-wheel-panel"
import { GestureGroupDialog } from "@/components/feature-dialogs"
import { GestureGroupPanel } from "@/components/gesture-group-panel"
import { KeyChord } from "@/components/key-chord"
import {
  AddButton,
  BindingDialog,
  LayerDialog,
  ShortcutDialog,
} from "@/components/rule-dialogs"
import { MouseBindingsPanel } from "@/components/trigger-features"
import { Button, buttonVariants } from "@/components/ui/button"
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Empty,
  EmptyHeader,
  EmptyTitle,
} from "@/components/ui/empty"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
} from "@/components/ui/sidebar"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { toast } from "@/components/ui/toast"
import {
  actionSummary,
  activeProfile,
  defaultGestureGroup,
  defaultLayer,
  normalizeConfig,
  triggerLabel,
  type UiConfig,
  type UiGestureGroup,
  type UiLayer,
  type UiProfile,
} from "@/lib/app-model"

type ShortcutDialogState =
  | { open: false }
  | { open: true; rule?: HotkeyRule }

type LayerDialogState =
  | { open: false }
  | { open: true; layer?: UiLayer }

type BindingDialogState =
  | { open: false }
  | { open: true; layerId: string; binding?: LayerBinding }

type GestureGroupDialogState =
  | { open: false }
  | { open: true; group: UiGestureGroup; isNew: boolean }

type NavigationTarget =
  | "general"
  | "edges"
  | "settings"
  | `layer:${string}`
  | `gesture:${string}`

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}

function App() {
  const [config, setConfig] = useState<UiConfig | null>(null)
  const [presets, setPresets] = useState<ActionPreset[]>([])
  const [configPath, setConfigPath] = useState("")
  const [startupEnabled, setStartupEnabled] = useState<boolean | null>(null)
  const [shortcutDialog, setShortcutDialog] = useState<ShortcutDialogState>({ open: false })
  const [layerDialog, setLayerDialog] = useState<LayerDialogState>({ open: false })
  const [bindingDialog, setBindingDialog] = useState<BindingDialogState>({ open: false })
  const [gestureGroupDialog, setGestureGroupDialog] =
    useState<GestureGroupDialogState>({ open: false })
  const [navigation, setNavigation] = useState<NavigationTarget>("general")
  const [layerTabs, setLayerTabs] = useState<Record<string, "keys" | "pointer">>({})
  const [showInitialLoading, setShowInitialLoading] = useState(false)
  const lastSavedRef = useRef("")
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const startupWriteRef = useRef(Promise.resolve())

  useEffect(() => {
    const loadingTimer = setTimeout(() => setShowInitialLoading(true), 500)

    Promise.all([
      AppService.GetConfig(),
      AppService.GetActionPresets(),
      AppService.GetConfigPath(),
    ])
      .then(([rawConfig, actionPresets, path]) => {
        const next = normalizeConfig(rawConfig)
        lastSavedRef.current = JSON.stringify(next)
        setConfig(next)
        setPresets(actionPresets ?? [])
        setConfigPath(path)
      })
      .catch((error) => {
        toast.add({
          type: "error",
          title: "加载失败",
          description: errorMessage(error),
        })
      })

    AppService.GetStartupEnabled()
      .then((enabled) => setStartupEnabled(enabled))
      .catch((error) => {
        setStartupEnabled(false)
        toast.add({
          type: "error",
          title: "读取开机自启失败",
          description: errorMessage(error),
        })
      })

    return () => clearTimeout(loadingTimer)
  }, [])

  useEffect(() => {
    return Events.On("config-changed", (event) => {
      const next = normalizeConfig(event.data)
      lastSavedRef.current = JSON.stringify(next)
      setConfig(next)
    })
  }, [])

  useEffect(() => {
    if (!config) {
      return
    }
    const serialized = JSON.stringify(config)
    if (serialized === lastSavedRef.current) {
      return
    }

    if (saveTimerRef.current) {
      clearTimeout(saveTimerRef.current)
    }
    saveTimerRef.current = setTimeout(() => {
      AppService.SaveConfig(config)
        .then((saved) => {
          const normalized = normalizeConfig(saved)
          lastSavedRef.current = JSON.stringify(normalized)
          setConfig(normalized)
        })
        .catch((error) => {
          toast.add({
            type: "error",
            title: "保存失败",
            description: errorMessage(error),
          })
        })
    }, 900)

    return () => {
      if (saveTimerRef.current) {
        clearTimeout(saveTimerRef.current)
      }
    }
  }, [config])

  const updateStartup = useCallback((enabled: boolean) => {
    setStartupEnabled(enabled)
    startupWriteRef.current = startupWriteRef.current.then(async () => {
      try {
        const actual = await AppService.SetStartupEnabled(enabled)
        setStartupEnabled((current) => (current === enabled ? actual : current))
      } catch (error) {
        setStartupEnabled((current) => (current === enabled ? !enabled : current))
        toast.add({
          type: "error",
          title: "开机自启设置失败",
          description: errorMessage(error),
        })
      }
    })
  }, [])

  const profile = useMemo(
    () => (config ? activeProfile(config) : undefined),
    [config],
  )

  const selectedLayer = useMemo(() => {
    if (!profile || !navigation.startsWith("layer:")) return undefined
    const id = navigation.slice("layer:".length)
    return profile.layers.find((layer) => layer.id === id)
  }, [navigation, profile])

  const selectedGesture = useMemo(() => {
    if (!profile || !navigation.startsWith("gesture:")) return undefined
    const id = navigation.slice("gesture:".length)
    return profile.gestures.find((group) => group.id === id)
  }, [navigation, profile])

  useEffect(() => {
    if (
      (navigation.startsWith("layer:") && profile && !selectedLayer) ||
      (navigation.startsWith("gesture:") && profile && !selectedGesture)
    ) {
      setNavigation("general")
    }
  }, [navigation, profile, selectedGesture, selectedLayer])

  const updateProfile = useCallback(
    (updater: (profile: UiProfile) => UiProfile) => {
      setConfig((current) => {
        if (!current) return current
        return {
          ...current,
          profiles: current.profiles.map((item) =>
            item.id === current.activeProfileId ? updater(item) : item,
          ),
        }
      })
    },
    [],
  )

  if (!config || !profile) {
    return showInitialLoading ? (
      <div className="flex h-full items-center justify-center">
        <Loader2Icon className="size-4 animate-spin text-muted-foreground" />
      </div>
    ) : null
  }

  const saveHotkey = (rule: HotkeyRule) => {
    updateProfile((current) => {
      const exists = current.hotkeys.some((item) => item.id === rule.id)
      return {
        ...current,
        hotkeys: exists
          ? current.hotkeys.map((item) => (item.id === rule.id ? rule : item))
          : [...current.hotkeys, rule],
      }
    })
  }

  const saveLayer = (layer: UiLayer) => {
    updateProfile((current) => {
      const exists = current.layers.some((item) => item.id === layer.id)
      return {
        ...current,
        layers: exists
          ? current.layers.map((item) => (item.id === layer.id ? layer : item))
          : [...current.layers, layer],
      }
    })
    setNavigation(`layer:${layer.id}`)
  }

  const saveBinding = (layerId: string, binding: LayerBinding) => {
    updateProfile((current) => ({
      ...current,
      layers: current.layers.map((layer) => {
        if (layer.id !== layerId) return layer
        const exists = layer.bindings.some((item) => item.id === binding.id)
        return {
          ...layer,
          bindings: exists
            ? layer.bindings.map((item) =>
                item.id === binding.id ? binding : item,
              )
            : [...layer.bindings, binding],
        }
      }),
    }))
  }

  const addLayer = () => {
    const used = new Set(profile.layers.map((layer) => layer.trigger))
    const trigger = [
      "CapsLock",
      "MouseRight",
      "MouseMiddle",
      "MouseBack",
      "MouseForward",
    ].find((candidate) => !used.has(candidate))
    if (!trigger) return

    const layer = defaultLayer(config.settings.defaultHoldMs)
    layer.trigger = trigger
    layer.name = triggerLabel(trigger)
    setLayerDialog({ open: true, layer })
  }

  const deleteLayer = (id: string) => {
    updateProfile((current) => ({
      ...current,
      layers: current.layers.filter((item) => item.id !== id),
    }))
    if (navigation === `layer:${id}`) {
      setNavigation("general")
    }
  }

  const availableGestureTriggers = [
    "MouseMiddle",
    "MouseRight",
    "MouseBack",
    "MouseForward",
  ].filter(
    (candidate) =>
      !profile.gestures.some((group) => group.trigger === candidate) &&
      !profile.layers.some(
        (layer) => layer.enabled && layer.trigger === candidate,
      ),
  )

  const addGestureGroup = () => {
    const trigger = availableGestureTriggers[0]
    if (!trigger) return

    setGestureGroupDialog({
      open: true,
      group: defaultGestureGroup(trigger),
      isNew: true,
    })
  }

  const saveGestureGroup = (group: UiGestureGroup) => {
    updateProfile((current) => {
      const exists = current.gestures.some((item) => item.id === group.id)
      return {
        ...current,
        gestures: exists
          ? current.gestures.map((item) =>
              item.id === group.id ? group : item,
            )
          : [...current.gestures, group],
      }
    })
    setNavigation(`gesture:${group.id}`)
  }

  const deleteGestureGroup = (id: string) => {
    updateProfile((current) => ({
      ...current,
      gestures: current.gestures.filter((item) => item.id !== id),
    }))
    if (navigation === `gesture:${id}`) {
      setNavigation("general")
    }
  }

  return (
    <SidebarProvider
      style={{ "--sidebar-width": "15rem" } as React.CSSProperties}
      className="h-svh min-h-0"
    >
      <Sidebar collapsible="none">
        <SidebarHeader className="border-b border-sidebar-border p-3">
          <div className="px-1 py-1 font-heading text-sm font-semibold">
            Key Flow
          </div>
        </SidebarHeader>

        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupLabel>快捷键</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    isActive={navigation === "general"}
                    onClick={() => setNavigation("general")}
                  >
                    <span>通用快捷键</span>
                  </SidebarMenuButton>
                  <SidebarMenuBadge>
                    {profile.hotkeysEnabled ? profile.hotkeys.length : "已禁用"}
                  </SidebarMenuBadge>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    isActive={navigation === "edges"}
                    onClick={() => setNavigation("edges")}
                  >
                    <span>屏幕边缘</span>
                  </SidebarMenuButton>
                  <SidebarMenuBadge>
                    {profile.edgeWheelsEnabled
                      ? profile.edgeWheels.length
                      : "已禁用"}
                  </SidebarMenuBadge>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

          <SidebarGroup>
            <SidebarGroupLabel>触发键</SidebarGroupLabel>
            <SidebarGroupAction
              type="button"
              aria-label="添加触发键"
              title={profile.layers.length >= 5 ? "已添加全部触发键" : "添加触发键"}
              disabled={profile.layers.length >= 5}
              onClick={addLayer}
            >
              <PlusIcon />
            </SidebarGroupAction>
            <SidebarGroupContent>
              <SidebarMenu>
                {profile.layers.map((layer) => (
                  <SidebarMenuItem key={layer.id}>
                    <SidebarMenuButton
                      isActive={navigation === `layer:${layer.id}`}
                      onClick={() => setNavigation(`layer:${layer.id}`)}
                    >
                      <span>{triggerLabel(layer.trigger)}</span>
                    </SidebarMenuButton>
                    <SidebarMenuBadge>
                      {layer.enabled
                        ? layer.bindings.length +
                          layer.mouseShortcuts.length +
                          (layer.windowOperations.enabled
                            ? (layer.windowOperations.moveButton ? 1 : 0) +
                              (layer.windowOperations.resizeButton ? 1 : 0) +
                              (layer.windowOperations.scaleWheel ? 1 : 0)
                            : 0)
                        : "已禁用"}
                    </SidebarMenuBadge>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

          <SidebarGroup>
            <SidebarGroupLabel>手势</SidebarGroupLabel>
            <SidebarGroupAction
              type="button"
              aria-label="添加手势启动键"
              title={
                availableGestureTriggers.length === 0
                  ? "没有可用的鼠标键"
                  : "添加手势启动键"
              }
              disabled={availableGestureTriggers.length === 0}
              onClick={addGestureGroup}
            >
              <PlusIcon />
            </SidebarGroupAction>
            <SidebarGroupContent>
              <SidebarMenu>
                {profile.gestures.map((group) => (
                  <SidebarMenuItem key={group.id}>
                    <SidebarMenuButton
                      isActive={navigation === `gesture:${group.id}`}
                      onClick={() => setNavigation(`gesture:${group.id}`)}
                    >
                      <span>{triggerLabel(group.trigger)}</span>
                    </SidebarMenuButton>
                    <SidebarMenuBadge>
                      {group.enabled ? group.bindings.length : "已禁用"}
                    </SidebarMenuBadge>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

        </SidebarContent>

        <SidebarFooter className="border-t border-sidebar-border p-2">
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                isActive={navigation === "settings"}
                onClick={() => setNavigation("settings")}
              >
                <SettingsIcon />
                <span>设置</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
          <div className="flex h-9 items-center justify-between px-2">
            <span className="text-sm">
              {config.enabled ? "已启用" : "已暂停"}
            </span>
            <Switch
              checked={config.enabled}
              onCheckedChange={(enabled) => setConfig({ ...config, enabled })}
              aria-label="启用全部快捷键"
            />
          </div>
        </SidebarFooter>
      </Sidebar>

      <SidebarInset className="h-svh min-h-0 overflow-hidden bg-muted/20">
        <ScrollArea className="min-h-0 flex-1">
          <div className="mx-auto w-full max-w-5xl p-6">
            {navigation === "general" && (
              <HotkeysPanel
                profile={profile}
                onEnabledChange={(hotkeysEnabled) =>
                  updateProfile((current) => ({
                    ...current,
                    hotkeysEnabled,
                  }))
                }
                onAdd={() => setShortcutDialog({ open: true })}
                onEdit={(rule) => setShortcutDialog({ open: true, rule })}
                onToggle={(id, enabled) =>
                  updateProfile((current) => ({
                    ...current,
                    hotkeys: current.hotkeys.map((item) =>
                      item.id === id ? { ...item, enabled } : item,
                    ),
                  }))
                }
                onDelete={(id) =>
                  updateProfile((current) => ({
                    ...current,
                    hotkeys: current.hotkeys.filter((item) => item.id !== id),
                  }))
                }
              />
            )}

            {selectedLayer && (
              <LayerPanel
                layer={selectedLayer}
                presets={presets}
                tab={layerTabs[selectedLayer.id] ?? "keys"}
                onTabChange={(tab) =>
                  setLayerTabs((current) => ({
                    ...current,
                    [selectedLayer.id]: tab,
                  }))
                }
                onChange={(next) =>
                  updateProfile((current) => ({
                    ...current,
                    layers: current.layers.map((item) =>
                      item.id === next.id ? next : item,
                    ),
                  }))
                }
                onEdit={() =>
                  setLayerDialog({ open: true, layer: selectedLayer })
                }
                onToggle={(enabled) =>
                  updateProfile((current) => ({
                    ...current,
                    layers: current.layers.map((item) =>
                      item.id === selectedLayer.id
                        ? { ...item, enabled }
                        : item,
                    ),
                  }))
                }
                onDelete={() => deleteLayer(selectedLayer.id)}
                onAddBinding={() =>
                  setBindingDialog({ open: true, layerId: selectedLayer.id })
                }
                onEditBinding={(binding) =>
                  setBindingDialog({
                    open: true,
                    layerId: selectedLayer.id,
                    binding,
                  })
                }
                onDeleteBinding={(bindingId) =>
                  updateProfile((current) => ({
                    ...current,
                    layers: current.layers.map((layer) =>
                      layer.id === selectedLayer.id
                        ? {
                            ...layer,
                            bindings: layer.bindings.filter(
                              (item) => item.id !== bindingId,
                            ),
                          }
                        : layer,
                    ),
                  }))
                }
              />
            )}

            {selectedGesture && (
              <GestureGroupPanel
                group={selectedGesture}
                presets={presets}
                onChange={saveGestureGroup}
                onEdit={() =>
                  setGestureGroupDialog({
                    open: true,
                    group: selectedGesture,
                    isNew: false,
                  })
                }
                onDelete={() => deleteGestureGroup(selectedGesture.id)}
              />
            )}

            {navigation === "edges" && (
              <EdgeWheelPanel
                enabled={profile.edgeWheelsEnabled}
                rules={profile.edgeWheels}
                onEnabledChange={(edgeWheelsEnabled) =>
                  updateProfile((current) => ({
                    ...current,
                    edgeWheelsEnabled,
                  }))
                }
                onChange={(edgeWheels) =>
                  updateProfile((current) => ({
                    ...current,
                    edgeWheels,
                  }))
                }
              />
            )}

            {navigation === "settings" && (
              <div className="flex flex-col gap-4">
                <h1 className="font-heading text-lg font-semibold">设置</h1>
                <SettingsPanel
                  config={config}
                  configPath={configPath}
                  startupEnabled={startupEnabled}
                  onStartupChange={updateStartup}
                  onChange={setConfig}
                />
              </div>
            )}
          </div>
        </ScrollArea>
      </SidebarInset>

      <ShortcutDialog
        open={shortcutDialog.open}
        rule={shortcutDialog.open ? shortcutDialog.rule : undefined}
        presets={presets}
        onClose={() => setShortcutDialog({ open: false })}
        onSave={saveHotkey}
      />
      <LayerDialog
        open={layerDialog.open}
        layer={layerDialog.open ? layerDialog.layer : undefined}
        defaultHoldMs={config.settings.defaultHoldMs}
        onClose={() => setLayerDialog({ open: false })}
        onSave={saveLayer}
      />
      <BindingDialog
        open={bindingDialog.open}
        binding={bindingDialog.open ? bindingDialog.binding : undefined}
        presets={presets}
        onClose={() => setBindingDialog({ open: false })}
        onSave={(binding) => {
          if (bindingDialog.open) {
            saveBinding(bindingDialog.layerId, binding)
          }
        }}
      />
      <GestureGroupDialog
        open={gestureGroupDialog.open}
        group={gestureGroupDialog.open ? gestureGroupDialog.group : undefined}
        isNew={gestureGroupDialog.open ? gestureGroupDialog.isNew : false}
        availableTriggers={availableGestureTriggers}
        onClose={() => setGestureGroupDialog({ open: false })}
        onSave={saveGestureGroup}
      />
    </SidebarProvider>
  )
}

function HotkeysPanel({
  profile,
  onEnabledChange,
  onAdd,
  onEdit,
  onToggle,
  onDelete,
}: {
  profile: UiProfile
  onEnabledChange: (enabled: boolean) => void
  onAdd: () => void
  onEdit: (rule: HotkeyRule) => void
  onToggle: (id: string, enabled: boolean) => void
  onDelete: (id: string) => void
}) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <h1 className="font-heading text-lg font-semibold">通用快捷键</h1>
          <Switch
            size="sm"
            checked={profile.hotkeysEnabled}
            onCheckedChange={onEnabledChange}
            aria-label="启用通用快捷键"
          />
        </div>
        <AddButton onClick={onAdd}>添加快捷键</AddButton>
      </div>
      <Card className="gap-0 py-0">
        <CardContent className="px-0">
          {profile.hotkeys.length === 0 ? (
            <Empty className="min-h-40">
              <EmptyHeader>
                <EmptyTitle>暂无快捷键</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : (
            profile.hotkeys.map((rule, index) => (
            <div key={rule.id}>
              {index > 0 && <Separator />}
              <div className="flex min-w-0 items-center gap-3 px-3 py-2">
                <div className="w-40 shrink-0">
                  <KeyChord keys={rule.keys} />
                </div>
                <div className="w-36 shrink-0 truncate text-sm font-medium">
                  {rule.name}
                </div>
                <div className="min-w-0 flex-1 truncate text-xs text-muted-foreground">
                  {actionSummary(rule.action)}
                </div>
                <Switch
                  size="sm"
                  checked={rule.enabled}
                  onCheckedChange={(enabled) => onToggle(rule.id, enabled)}
                  aria-label={"启用 " + rule.name}
                />
                <RowMenu
                  onEdit={() => onEdit(rule)}
                  onDelete={() => onDelete(rule.id)}
                />
              </div>
            </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function LayerPanel({
  layer,
  presets,
  tab,
  onTabChange,
  onChange,
  onEdit,
  onToggle,
  onDelete,
  onAddBinding,
  onEditBinding,
  onDeleteBinding,
}: {
  layer: UiLayer
  presets: ActionPreset[]
  tab: "keys" | "pointer"
  onTabChange: (tab: "keys" | "pointer") => void
  onChange: (layer: UiLayer) => void
  onEdit: () => void
  onToggle: (enabled: boolean) => void
  onDelete: () => void
  onAddBinding: () => void
  onEditBinding: (binding: LayerBinding) => void
  onDeleteBinding: (bindingId: string) => void
}) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex items-center gap-3">
            <h1 className="truncate font-heading text-lg font-semibold">
              {triggerLabel(layer.trigger)}
            </h1>
            <Switch
              size="sm"
              checked={layer.enabled}
              onCheckedChange={onToggle}
              aria-label={"启用 " + triggerLabel(layer.trigger)}
            />
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Button type="button" variant="outline" size="sm" onClick={onEdit}>
            设置
          </Button>
          <RowMenu onDelete={onDelete} />
        </div>
      </div>

      <Tabs
        value={tab}
        onValueChange={(value) => onTabChange(value as "keys" | "pointer")}
        className="gap-4"
      >
        <div className="flex items-center justify-between gap-3">
          <TabsList className="grid w-full max-w-md grid-cols-2">
            <TabsTrigger value="keys">按键映射</TabsTrigger>
            <TabsTrigger value="pointer">鼠标</TabsTrigger>
          </TabsList>
          {tab === "keys" && (
            <AddButton onClick={onAddBinding}>添加映射</AddButton>
          )}
        </div>

        <TabsContent value="keys">
          <Card className="gap-0 py-0">
            <CardContent className="px-0">
              {layer.bindings.length === 0 ? (
                <Empty className="min-h-40">
                  <EmptyHeader>
                    <EmptyTitle>暂无映射</EmptyTitle>
                  </EmptyHeader>
                </Empty>
              ) : (
                layer.bindings.map((binding, index) => (
                  <div key={binding.id}>
                    {index > 0 && <Separator />}
                    <div className="flex min-w-0 items-center gap-3 px-3 py-2">
                      <div className="w-36 shrink-0">
                        <KeyChord keys={binding.keys} />
                      </div>
                      <div className="min-w-0 flex-1 truncate text-sm text-muted-foreground">
                        {actionSummary(binding.action)}
                      </div>
                      <RowMenu
                        onEdit={() => onEditBinding(binding)}
                        onDelete={() => onDeleteBinding(binding.id)}
                      />
                    </div>
                  </div>
                ))
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="pointer">
          <MouseBindingsPanel
            layer={layer}
            presets={presets}
            onChange={onChange}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}

function SettingsPanel({
  config,
  configPath,
  startupEnabled,
  onStartupChange,
  onChange,
}: {
  config: UiConfig
  configPath: string
  startupEnabled: boolean | null
  onStartupChange: (enabled: boolean) => void
  onChange: (config: UiConfig) => void
}) {
  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>启动</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-4">
            <div className="text-sm font-medium">开机自启</div>
            <Switch
              checked={startupEnabled ?? false}
              disabled={startupEnabled === null}
              onCheckedChange={onStartupChange}
              aria-label="开机自启"
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>触发键</CardTitle>
        </CardHeader>
        <CardContent>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="default-hold">默认长按时间</FieldLabel>
              <Input
                id="default-hold"
                type="number"
                min={50}
                max={1500}
                value={config.settings.defaultHoldMs}
                onChange={(event) =>
                  onChange({
                    ...config,
                    settings: {
                      ...config.settings,
                      defaultHoldMs: Number(event.target.value) || 200,
                    },
                  })
                }
              />
              <FieldDescription>50–1500 ms</FieldDescription>
            </Field>
          </FieldGroup>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>手势</CardTitle>
        </CardHeader>
        <CardContent>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="gesture-timeout">无操作超时</FieldLabel>
              <Input
                id="gesture-timeout"
                type="number"
                min={500}
                max={10000}
                step={100}
                value={config.settings.gestureTimeoutMs}
                onChange={(event) =>
                  onChange({
                    ...config,
                    settings: {
                      ...config.settings,
                      gestureTimeoutMs: Number(event.target.value) || 1000,
                    },
                  })
                }
              />
              <FieldDescription>
                500–10000 ms；手势期间持续无移动达到该时间后直接取消
              </FieldDescription>
            </Field>
          </FieldGroup>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>数据</CardTitle>
          <CardAction>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() =>
                void AppService.OpenConfigDirectory().catch((error) => {
                  toast.add({
                    type: "error",
                    title: "打开目录失败",
                    description: errorMessage(error),
                  })
                })
              }
            >
              <FolderOpenIcon data-icon="inline-start" />
              打开目录
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          <div className="space-y-1.5 rounded-lg border px-4 py-3">
            <div className="text-xs font-medium">配置文件</div>
            <code className="block min-w-0 select-text break-all font-mono text-xs leading-relaxed text-muted-foreground">
              {configPath}
            </code>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function RowMenu({
  onEdit,
  onDelete,
}: {
  onEdit?: () => void
  onDelete: () => void
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={buttonVariants({
          variant: "ghost",
          size: "icon-sm",
          className:
            "data-popup-open:bg-accent data-popup-open:text-accent-foreground",
        })}
        aria-label="更多操作"
      >
        <MoreHorizontalIcon />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-32">
        <DropdownMenuGroup>
          {onEdit && (
            <DropdownMenuItem onClick={onEdit}>
              <PencilIcon />
              编辑
            </DropdownMenuItem>
          )}
          <DropdownMenuItem variant="destructive" onClick={onDelete}>
            <Trash2Icon />
            删除
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export default App
