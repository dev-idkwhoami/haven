export interface LocalProfile {
  id: number;
  publicKey: string;
  displayName: string;
  avatar: string | null;
  avatarHash: string | null;
  bio: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface TrustedServer {
  id: number;
  address: string;
  publicKey: string;
  name: string | null;
  icon: string | null;
  iconHash: string | null;
  sessionToken: string | null;
  isRelayOnly: boolean;
  firstTrustedAt: string;
  lastConnectedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface ServerInfo {
  id: string;
  name: string;
  description: string | null;
  icon: string | null;
  iconHash: string;
  accessMode: string;
  maxFileSize: number;
  totalStorageLimit: number;
  defaultChannelId: string | null;
  welcomeMessage: string | null;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface Category {
  id: string;
  serverId: number;
  remoteCategoryId: string;
  name: string;
  position: number;
  type: string;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface Channel {
  id: string;
  serverId: number;
  remoteChannelId: string;
  remoteCategoryId: string;
  name: string;
  type: "text" | "voice";
  position: number;
  lastReadMessageId: string | null;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface Message {
  id: string;
  serverId: number;
  remoteMessageId: string;
  channelId: string;
  authorPubKey: string;
  content: string;
  signature: string;
  nonce: string;
  editedAt: string | null;
  remoteCreatedAt: string;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface User {
  id: string;
  serverId: number;
  publicKey: string;
  displayName: string;
  avatar: string | null;
  avatarHash: string | null;
  bio: string | null;
  status?: string;
  roleIds?: string[];
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface Role {
  id: string;
  name: string;
  color: string | null;
  position: number;
  isDefault: boolean;
  permissions: number;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface DMConversation {
  id: string;
  isGroup: boolean;
  name: string | null;
  createdBy: string | null;
  participants: DMParticipant[];
  createdAt: string;
  updatedAt: string;
}

export interface DMParticipant {
  conversationId: string;
  userId: string;
  isKeyManager: boolean;
  joinedAt: string;
  leftAt: string | null;
}

export interface DMMessage {
  id: string;
  conversationId: string;
  senderPubKey: string;
  content: string;
  remoteCreatedAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface PerServerConfig {
  id: number;
  serverId: number;
  syncAvatars: boolean;
  syncBios: boolean;
  syncStatus: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface AppSettings {
  inputDeviceId: string;
  outputDeviceId: string;
  inputVolume: number;
  outputVolume: number;
  pushToTalk: boolean;
  pushToTalkKey: string;
  noiseSuppression: boolean;
  theme: string;
  fontSize: number;
  compactMode: boolean;
}

export interface AuditLogEntry {
  id: string;
  action: string;
  actorPubKey: string;
  targetType: string;
  targetId: string;
  details: string | null;
  createdAt: string;
}

export interface Invite {
  id: string;
  code: string;
  createdBy: string;
  maxUses: number;
  useCount: number;
  expiresAt: string | null;
  createdAt: string;
}

export interface AccessRequest {
  id: string;
  pubKey: string;
  displayName: string;
  message: string;
  isOnline: boolean;
  createdAt: string;
}

export interface VoiceParticipant {
  publicKey: string;
  displayName: string;
  isMuted: boolean;
  isDeafened: boolean;
  isSpeaking: boolean;
}

export interface AudioDevice {
  id: string;
  name: string;
  isDefault: boolean;
}

export type AppPhase = "loading" | "setup" | "ready";

export type ConnectionStatus = "disconnected" | "connecting" | "authenticating" | "connected" | "reconnecting" | "error";

export interface AppState {
  phase: AppPhase;
  loadingMsg: string;
  progress: number;
}

export interface FileProgress {
  fileId: string;
  fileName: string;
  progress: number;
  total: number;
}

export type TrustStatus = "new" | "trusted" | "mismatch";

export interface ServerHello {
  serverPubKey: string;
  serverName: string;
  accessMode: string;
  trustStatus: TrustStatus;
  storedPubKey: string;
}

export interface ServerEntry {
  id: number;
  address: string;
  name: string;
  iconHash: string;
  isRelayOnly: boolean;
  isOwner: boolean;
  connected: boolean;
  lastConnectedAt: string;
}
