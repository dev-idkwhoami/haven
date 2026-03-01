import type { Category, Channel } from "../types";
import { ChannelService, on } from "../wails";

let _categories = $state<Category[]>([]);
let _channels = $state<Channel[]>([]);
let _activeChannelId = $state<string | null>(null);

export function categories() { return _categories; }
export function channels() { return _channels; }
export function activeChannelId() { return _activeChannelId; }

on("channel:created", (data: unknown) => {
  const ch = data as Channel;
  _channels = [..._channels, ch];
});

on("channel:updated", (data: unknown) => {
  const ch = data as Channel;
  _channels = _channels.map((c) => (c.remoteChannelId === ch.remoteChannelId ? ch : c));
});

on("channel:deleted", (data: unknown) => {
  const d = data as { channelId: string };
  _channels = _channels.filter((c) => c.remoteChannelId !== d.channelId);
  if (_activeChannelId === d.channelId) {
    _activeChannelId = null;
  }
});

on("category:created", (data: unknown) => {
  const cat = data as Category;
  _categories = [..._categories, cat];
});

on("category:updated", (data: unknown) => {
  const cat = data as Category;
  _categories = _categories.map((c) => (c.remoteCategoryId === cat.remoteCategoryId ? cat : c));
});

on("category:deleted", (data: unknown) => {
  const d = data as { categoryId: string };
  _categories = _categories.filter((c) => c.remoteCategoryId !== d.categoryId);
  _channels = _channels.filter((c) => c.remoteCategoryId !== d.categoryId);
});

export async function loadChannels(serverId: number): Promise<void> {
  try {
    _categories = await ChannelService.GetCategories(serverId);
    _channels = await ChannelService.GetChannels(serverId);
  } catch {
    _categories = [];
    _channels = [];
  }
}

export async function createCategory(serverId: number, name: string, type: string): Promise<void> {
  await ChannelService.CreateCategory(serverId, name, type);
}

export async function updateCategory(
  serverId: number,
  categoryId: string,
  name: string,
  position: number,
): Promise<void> {
  await ChannelService.UpdateCategory(serverId, categoryId, name, position);
}

export async function deleteCategory(serverId: number, categoryId: string): Promise<void> {
  await ChannelService.DeleteCategory(serverId, categoryId);
}

export async function createChannel(
  serverId: number,
  categoryId: string,
  name: string,
  type: string,
): Promise<void> {
  await ChannelService.CreateChannel(serverId, categoryId, name, type);
}

export async function updateChannel(
  serverId: number,
  channelId: string,
  name: string,
  position: number,
): Promise<void> {
  await ChannelService.UpdateChannel(serverId, channelId, name, position);
}

export async function deleteChannel(serverId: number, channelId: string): Promise<void> {
  await ChannelService.DeleteChannel(serverId, channelId);
}

export function setActiveChannel(channelId: string | null): void {
  _activeChannelId = channelId;
}
