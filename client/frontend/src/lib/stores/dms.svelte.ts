import type { DMConversation, DMMessage } from "../types";
import { DMService, on } from "../wails";

let _conversations = $state<DMConversation[]>([]);
let _activeDMId = $state<string | null>(null);
let _dmMessages = $state<DMMessage[]>([]);
let _incomingCallId = $state<string | null>(null);
let _activeCallId = $state<string | null>(null);

export function conversations() { return _conversations; }
export function activeDMId() { return _activeDMId; }
export function dmMessages() { return _dmMessages; }
export function incomingCallId() { return _incomingCallId; }
export function activeCallId() { return _activeCallId; }

on("dm:created", (data: unknown) => {
  const conv = data as DMConversation;
  if (!_conversations.some((c) => c.id === conv.id)) {
    _conversations = [..._conversations, conv];
  }
});

on("dm:message", (data: unknown) => {
  const msg = data as DMMessage;
  if (msg.conversationId === _activeDMId) {
    _dmMessages = [..._dmMessages, msg];
  }
});

on("dm:memberAdded", (data: unknown) => {
  const d = data as { conversationId: string; participant: { userId: string } };
  _conversations = _conversations.map((c) => {
    if (c.id === d.conversationId) {
      return {
        ...c,
        participants: [...c.participants, { conversationId: d.conversationId, userId: d.participant.userId, isKeyManager: false, joinedAt: new Date().toISOString(), leftAt: null }],
      };
    }
    return c;
  });
});

on("dm:memberRemoved", (data: unknown) => {
  const d = data as { conversationId: string; userId: string };
  _conversations = _conversations.map((c) => {
    if (c.id === d.conversationId) {
      return { ...c, participants: c.participants.filter((p) => p.userId !== d.userId) };
    }
    return c;
  });
});

on("dm:callIncoming", (data: unknown) => {
  const d = data as { conversationId: string };
  _incomingCallId = d.conversationId;
});

on("dm:callStarted", (data: unknown) => {
  const d = data as { conversationId: string };
  _activeCallId = d.conversationId;
  _incomingCallId = null;
});

on("dm:callEnded", () => {
  _activeCallId = null;
  _incomingCallId = null;
});

export async function loadConversations(): Promise<void> {
  try {
    _conversations = await DMService.GetConversations();
  } catch {
    _conversations = [];
  }
}

export async function createDM(participantKeys: string[]): Promise<void> {
  await DMService.CreateDM(participantKeys);
}

export async function sendDM(conversationId: string, content: string): Promise<void> {
  await DMService.Send(conversationId, content);
}

export async function loadDMHistory(conversationId: string, before: string = "", limit: number = 50): Promise<void> {
  try {
    const batch = await DMService.GetHistory(conversationId, before, limit);
    _dmMessages = [...batch, ..._dmMessages];
  } catch {
    // History not available
  }
}

export function setActiveDM(conversationId: string | null): void {
  _activeDMId = conversationId;
  _dmMessages = [];
}

export async function addMember(conversationId: string, publicKey: string): Promise<void> {
  await DMService.AddMember(conversationId, publicKey);
}

export async function removeMember(conversationId: string, publicKey: string): Promise<void> {
  await DMService.RemoveMember(conversationId, publicKey);
}

export async function leaveConversation(conversationId: string): Promise<void> {
  await DMService.LeaveConversation(conversationId);
  _conversations = _conversations.filter((c) => c.id !== conversationId);
  if (_activeDMId === conversationId) {
    _activeDMId = null;
    _dmMessages = [];
  }
}

export async function startCall(conversationId: string): Promise<void> {
  await DMService.StartCall(conversationId);
}

export async function acceptCall(conversationId: string): Promise<void> {
  await DMService.AcceptCall(conversationId);
}

export async function rejectCall(conversationId: string): Promise<void> {
  await DMService.RejectCall(conversationId);
  _incomingCallId = null;
}

export async function leaveCall(conversationId: string): Promise<void> {
  await DMService.LeaveCall(conversationId);
  _activeCallId = null;
}
