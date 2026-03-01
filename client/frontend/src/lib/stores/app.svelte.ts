import type { AppPhase, AppState } from "../types";
import { AppService, on } from "../wails";

let _phase = $state<AppPhase>("loading");
let _loadingMsg = $state("Initializing...");
let _progress = $state(0);

export function phase() { return _phase; }
export function loadingMsg() { return _loadingMsg; }
export function progress() { return _progress; }

on("app:stateChanged", (data: unknown) => {
  const state = data as AppState;
  _phase = state.phase;
  _loadingMsg = state.loadingMsg;
  _progress = state.progress;
});

export async function init(): Promise<void> {
  try {
    const state = await AppService.GetState();
    _phase = state.phase;
    _loadingMsg = state.loadingMsg;
    _progress = state.progress;
  } catch {
    // Backend not ready yet — stay in loading phase
  }
}

export async function shutdown(): Promise<void> {
  await AppService.Shutdown();
}
