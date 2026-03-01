import type { Message } from "../types";
import { MessageService, on } from "../wails";

let _messages = $state<Message[]>([]);
let _typingUsers = $state<string[]>([]);
let _hasMore = $state(true);
let _loading = $state(false);

export function messages() { return _messages; }
export function typingUsers() { return _typingUsers; }
export function hasMore() { return _hasMore; }
export function loading() { return _loading; }

on("message:new", (data: unknown) => {
  const msg = data as Message;
  _messages = [..._messages, msg];
});

on("message:edited", (data: unknown) => {
  const msg = data as Message;
  _messages = _messages.map((m) => (m.remoteMessageId === msg.remoteMessageId ? msg : m));
});

on("message:deleted", (data: unknown) => {
  const d = data as { messageId: string };
  _messages = _messages.filter((m) => m.remoteMessageId !== d.messageId);
});

export function clearMessages(): void {
  _messages = [];
  _hasMore = true;
}

export async function loadHistory(serverId: number, channelId: string, before: string = "", limit: number = 50): Promise<void> {
  if (_loading) return;
  _loading = true;
  try {
    const batch = await MessageService.GetHistory(serverId, channelId, before, limit);
    if (batch.length < limit) {
      _hasMore = false;
    }
    _messages = [...batch, ..._messages];
  } finally {
    _loading = false;
  }
}

export async function sendMessage(serverId: number, channelId: string, content: string): Promise<void> {
  await MessageService.Send(serverId, channelId, content);
}

export async function editMessage(serverId: number, messageId: string, content: string): Promise<void> {
  await MessageService.Edit(serverId, messageId, content);
}

export async function deleteMessage(serverId: number, messageId: string): Promise<void> {
  await MessageService.Delete(serverId, messageId);
}

export async function search(serverId: number, channelId: string, query: string): Promise<Message[]> {
  return MessageService.Search(serverId, channelId, query);
}
