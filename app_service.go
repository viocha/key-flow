package main

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type AppService struct {
	store           *ConfigStore
	engine          *InputEngine
	app             *application.App
	onConfigChanged func(AppConfig)
}

func NewAppService(store *ConfigStore, engine *InputEngine) *AppService {
	return &AppService{store: store, engine: engine}
}

func (s *AppService) GetConfig() AppConfig {
	return s.store.Snapshot()
}

func (s *AppService) SaveConfig(config AppConfig) (AppConfig, error) {
	saved, err := s.store.Save(config)
	if err != nil {
		return AppConfig{}, err
	}
	if err := s.engine.UpdateConfig(saved); err != nil {
		return AppConfig{}, err
	}
	if s.app != nil {
		s.app.Event.Emit("config-changed", saved)
	}
	if s.onConfigChanged != nil {
		s.onConfigChanged(saved)
	}
	return saved, nil
}

func (s *AppService) GetStatus() EngineStatus {
	return s.engine.Status()
}

func (s *AppService) GetConfigPath() string {
	return s.store.Path()
}

func (s *AppService) OpenConfigDirectory() error {
	directory := filepath.Dir(s.store.Path())
	if err := exec.Command("explorer.exe", directory).Start(); err != nil {
		return fmt.Errorf("打开配置目录失败: %w", err)
	}
	return nil
}

func (s *AppService) GetActionPresets() []ActionPreset {
	return actionPresets()
}

func (s *AppService) GetStartupEnabled() (bool, error) {
	return startupTaskEnabled()
}

func (s *AppService) SetStartupEnabled(enabled bool) (bool, error) {
	if err := setStartupTaskEnabled(enabled); err != nil {
		return false, err
	}
	return startupTaskEnabled()
}
