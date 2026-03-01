import type { AppSettings, PerServerConfig } from "../types";
import { SettingsService, on } from "../wails";

let _appSettings = $state<AppSettings | null>(null);
let _perServerSettings = $state<PerServerConfig | null>(null);

export function appSettings() { return _appSettings; }
export function perServerSettings() { return _perServerSettings; }

on("settings:appChanged", (data: unknown) => {
  _appSettings = data as AppSettings;
});

on("settings:serverChanged", (data: unknown) => {
  _perServerSettings = data as PerServerConfig;
});

export async function loadAppSettings(): Promise<void> {
  try {
    _appSettings = await SettingsService.GetAppSettings();
  } catch {
    // Not available yet
  }
}

export async function updateAppSettings(settings: Partial<AppSettings>): Promise<void> {
  _appSettings = await SettingsService.UpdateAppSettings(settings);
}

export async function loadServerSettings(serverId: number): Promise<void> {
  try {
    _perServerSettings = await SettingsService.GetServerSettings(serverId);
  } catch {
    _perServerSettings = null;
  }
}

export async function updateServerSettings(
  serverId: number,
  settings: Partial<PerServerConfig>,
): Promise<void> {
  _perServerSettings = await SettingsService.UpdateServerSettings(serverId, settings);
}
