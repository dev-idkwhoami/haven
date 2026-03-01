import type { VoiceParticipant, AudioDevice } from "../types";
import { VoiceService, on } from "../wails";

let _activeVoiceChannelId = $state<string | null>(null);
let _participants = $state<VoiceParticipant[]>([]);
let _isMuted = $state(false);
let _isDeafened = $state(false);
let _inputDevices = $state<AudioDevice[]>([]);
let _outputDevices = $state<AudioDevice[]>([]);
let _volume = $state(100);

export function activeVoiceChannelId() { return _activeVoiceChannelId; }
export function participants() { return _participants; }
export function isMuted() { return _isMuted; }
export function isDeafened() { return _isDeafened; }
export function inputDevices() { return _inputDevices; }
export function outputDevices() { return _outputDevices; }
export function volume() { return _volume; }

on("voice:joined", (data: unknown) => {
  const p = data as VoiceParticipant;
  if (!_participants.some((x) => x.publicKey === p.publicKey)) {
    _participants = [..._participants, p];
  }
});

on("voice:left", (data: unknown) => {
  const d = data as { publicKey: string };
  _participants = _participants.filter((p) => p.publicKey !== d.publicKey);
});

on("voice:muted", (data: unknown) => {
  const d = data as { publicKey: string; muted: boolean };
  _participants = _participants.map((p) =>
    p.publicKey === d.publicKey ? { ...p, isMuted: d.muted } : p,
  );
});

on("voice:deafened", (data: unknown) => {
  const d = data as { publicKey: string; deafened: boolean };
  _participants = _participants.map((p) =>
    p.publicKey === d.publicKey ? { ...p, isDeafened: d.deafened } : p,
  );
});

on("voice:speaking", (data: unknown) => {
  const d = data as { publicKey: string; speaking: boolean };
  _participants = _participants.map((p) =>
    p.publicKey === d.publicKey ? { ...p, isSpeaking: d.speaking } : p,
  );
});

export async function joinChannel(serverId: number, channelId: string): Promise<void> {
  await VoiceService.JoinChannel(serverId, channelId);
  _activeVoiceChannelId = channelId;
}

export async function leaveChannel(): Promise<void> {
  await VoiceService.LeaveChannel();
  _activeVoiceChannelId = null;
  _participants = [];
}

export async function setMuted(muted: boolean): Promise<void> {
  await VoiceService.SetMuted(muted);
  _isMuted = muted;
}

export async function setDeafened(deafened: boolean): Promise<void> {
  await VoiceService.SetDeafened(deafened);
  _isDeafened = deafened;
}

export async function loadDevices(): Promise<void> {
  try {
    _inputDevices = await VoiceService.GetInputDevices();
    _outputDevices = await VoiceService.GetOutputDevices();
    _volume = await VoiceService.GetVolume();
  } catch {
    // Devices not available
  }
}

export async function setInputDevice(deviceId: string): Promise<void> {
  await VoiceService.SetInputDevice(deviceId);
}

export async function setOutputDevice(deviceId: string): Promise<void> {
  await VoiceService.SetOutputDevice(deviceId);
}

export async function setVolume(v: number): Promise<void> {
  await VoiceService.SetVolume(v);
  _volume = v;
}
