// --- Wails Runtime Events ---
import { EventsOn, EventsOff, EventsEmit } from "./wailsjs/runtime/runtime";

// --- Generated Service Bindings ---
import * as AppServiceBinding from "./wailsjs/go/services/AppService";
import * as ProfileServiceBinding from "./wailsjs/go/services/ProfileService";
import * as ServerServiceBinding from "./wailsjs/go/services/ServerService";
import * as ChannelServiceBinding from "./wailsjs/go/services/ChannelService";
import * as MessageServiceBinding from "./wailsjs/go/services/MessageService";
import * as UserServiceBinding from "./wailsjs/go/services/UserService";
import * as RoleServiceBinding from "./wailsjs/go/services/RoleService";
import * as VoiceServiceBinding from "./wailsjs/go/services/VoiceService";
import * as DMServiceBinding from "./wailsjs/go/services/DMService";
import * as FileServiceBinding from "./wailsjs/go/services/FileService";
import * as SettingsServiceBinding from "./wailsjs/go/services/SettingsService";
import * as AdminServiceBinding from "./wailsjs/go/services/AdminService";

// --- Event helpers ---
type EventCallback = (...args: unknown[]) => void;

export function on(eventName: string, callback: EventCallback): () => void {
  return EventsOn(eventName, callback);
}

export function off(eventName: string): void {
  EventsOff(eventName);
}

export function emit(eventName: string, ...data: unknown[]): void {
  EventsEmit(eventName, ...data);
}

// --- Service re-exports ---
export const AppService = AppServiceBinding;
export const ProfileService = ProfileServiceBinding;
export const ServerService = ServerServiceBinding;
export const ChannelService = ChannelServiceBinding;
export const MessageService = MessageServiceBinding;
export const UserService = UserServiceBinding;
export const RoleService = RoleServiceBinding;
export const VoiceService = VoiceServiceBinding;
export const DMService = DMServiceBinding;
export const FileService = FileServiceBinding;
export const SettingsService = SettingsServiceBinding;
export const AdminService = AdminServiceBinding;
