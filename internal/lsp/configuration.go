package lsp

import (
	"context"
	"encoding/json"

	"go.lsp.dev/protocol"
	"go.yorun.ai/skelc/internal/lsp/analysis"
)

type _SchemaCompatibilitySettings struct {
	Diagnostics       bool   `json:"diagnostics"`
	IncludeCompatible bool   `json:"includeCompatible"`
	CodeLens          bool   `json:"codeLens"`
	Baseline          string `json:"baseline"`
}

type _InitializationOptions struct {
	SchemaCompatibility _SchemaCompatibilitySettings `json:"schemaCompatibility"`
}

type _SchemaCompatibilitySettingsPatch struct {
	Diagnostics       *bool   `json:"diagnostics"`
	IncludeCompatible *bool   `json:"includeCompatible"`
	CodeLens          *bool   `json:"codeLens"`
	Baseline          *string `json:"baseline"`
}

func defaultSchemaCompatibilitySettings() _SchemaCompatibilitySettings {
	return _SchemaCompatibilitySettings{CodeLens: true}
}

func decodeInitializationSettings(value protocol.LSPAny) _SchemaCompatibilitySettings {
	settings := defaultSchemaCompatibilitySettings()
	if len(value) == 0 {
		return settings
	}
	options := _InitializationOptions{SchemaCompatibility: settings}
	if json.Unmarshal(value, &options) == nil {
		return options.SchemaCompatibility
	}
	return settings
}

func decodeChangedSettings(value protocol.LSPAny, fallback _SchemaCompatibilitySettings) _SchemaCompatibilitySettings {
	if len(value) == 0 {
		return fallback
	}
	var envelope struct {
		Skelc struct {
			SchemaCompatibility *_SchemaCompatibilitySettingsPatch `json:"schemaCompatibility"`
		} `json:"skelc"`
		SchemaCompatibility *_SchemaCompatibilitySettingsPatch `json:"schemaCompatibility"`
	}
	if json.Unmarshal(value, &envelope) != nil {
		return fallback
	}
	if envelope.Skelc.SchemaCompatibility != nil {
		return applySchemaCompatibilitySettings(fallback, *envelope.Skelc.SchemaCompatibility)
	}
	if envelope.SchemaCompatibility != nil {
		return applySchemaCompatibilitySettings(fallback, *envelope.SchemaCompatibility)
	}
	return fallback
}

func applySchemaCompatibilitySettings(settings _SchemaCompatibilitySettings, patch _SchemaCompatibilitySettingsPatch) _SchemaCompatibilitySettings {
	if patch.Diagnostics != nil {
		settings.Diagnostics = *patch.Diagnostics
	}
	if patch.IncludeCompatible != nil {
		settings.IncludeCompatible = *patch.IncludeCompatible
	}
	if patch.CodeLens != nil {
		settings.CodeLens = *patch.CodeLens
	}
	if patch.Baseline != nil {
		settings.Baseline = *patch.Baseline
	}
	return settings
}

func (s *_Server) DidChangeConfiguration(ctx context.Context, params *protocol.DidChangeConfigurationParams) error {
	s.mu.Lock()
	previous := s.schemaCompatibility
	s.schemaCompatibility = decodeChangedSettings(params.Settings, previous)
	changed := previous != s.schemaCompatibility
	client := s.client
	refreshCodeLens := s.codeLensRefreshSupport
	s.mu.Unlock()
	if !changed {
		return nil
	}
	s.invalidateSemanticDiagnostics(ctx)
	if client != nil && refreshCodeLens {
		_ = client.CodeLensRefresh(ctx)
	}
	return nil
}

func (s *_Server) compatibilityAnalysisOptions() analysis.CompatibilityOptions {
	s.mu.RLock()
	settings := s.schemaCompatibility
	s.mu.RUnlock()
	return analysis.CompatibilityOptions{
		Enabled: settings.Diagnostics, IncludeCompatible: settings.IncludeCompatible, BaselineSkelIn: settings.Baseline,
	}
}
