package services

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"haven/client/connection"
	"haven/shared"
)

// Category is the frontend-facing category data.
type Category struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
	Type     string `json:"type"`
}

// Channel is the frontend-facing channel data.
type Channel struct {
	ID          string   `json:"id"`
	CategoryID  string   `json:"categoryId"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Position    int      `json:"position"`
	OpusBitrate int      `json:"opusBitrate"`
	RoleIDs     []string `json:"roleIds"`
}

// ChannelService manages channel and category operations.
type ChannelService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey
}

// NewChannelService creates a new ChannelService.
func NewChannelService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *ChannelService {
	return &ChannelService{
		db:      db,
		manager: manager,
		privKey: privKey,
	}
}

// SetContext is called by Wails during startup.
func (c *ChannelService) SetContext(ctx context.Context) {
	c.ctx = ctx
}

// GetCategories returns all categories for a server.
func (c *ChannelService) GetCategories(serverID int64) ([]Category, error) {
	conn, err := c.manager.Get(serverID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	resp, err := conn.Request(shared.TypeCategoryList, nil)
	if err != nil {
		return nil, fmt.Errorf("category.list: %w", err)
	}

	var result struct {
		Categories []Category `json:"categories"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("unmarshal categories: %w", err)
	}
	return result.Categories, nil
}

// CreateCategory creates a new category on a server.
func (c *ChannelService) CreateCategory(serverID int64, name string, typ string) (Category, error) {
	conn, err := c.manager.Get(serverID)
	if err != nil {
		return Category{}, fmt.Errorf("get connection: %w", err)
	}

	resp, err := conn.Request(shared.TypeCategoryCreate, map[string]interface{}{
		"name": name,
		"type": typ,
	})
	if err != nil {
		return Category{}, fmt.Errorf("category.create: %w", err)
	}

	var created struct {
		ID       string `json:"id"`
		Position int    `json:"position"`
		Version  int64  `json:"version"`
	}
	if err := json.Unmarshal(resp.Payload, &created); err != nil {
		return Category{}, fmt.Errorf("unmarshal category: %w", err)
	}

	return Category{
		ID:       created.ID,
		Name:     name,
		Position: created.Position,
		Type:     typ,
	}, nil
}

// UpdateCategory updates a category on a server.
func (c *ChannelService) UpdateCategory(serverID int64, id string, name string, position int) error {
	conn, err := c.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeCategoryUpdate, map[string]interface{}{
		"id":       id,
		"name":     name,
		"position": position,
	})
	if err != nil {
		return fmt.Errorf("category.update: %w", err)
	}
	return nil
}

// DeleteCategory deletes a category and all its channels.
func (c *ChannelService) DeleteCategory(serverID int64, id string) error {
	conn, err := c.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeCategoryDelete, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return fmt.Errorf("category.delete: %w", err)
	}
	return nil
}

// GetChannels returns channels for a server, optionally filtered by category.
func (c *ChannelService) GetChannels(serverID int64, categoryID string) ([]Channel, error) {
	conn, err := c.manager.Get(serverID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{}
	if categoryID != "" {
		payload["category_id"] = categoryID
	}

	resp, err := conn.Request(shared.TypeChannelList, payload)
	if err != nil {
		return nil, fmt.Errorf("channel.list: %w", err)
	}

	var result struct {
		Channels []channelWS `json:"channels"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("unmarshal channels: %w", err)
	}

	channels := make([]Channel, len(result.Channels))
	for i, ch := range result.Channels {
		channels[i] = ch.toChannel()
	}
	return channels, nil
}

// CreateChannel creates a new channel on a server.
func (c *ChannelService) CreateChannel(serverID int64, categoryID string, name string, typ string, roleIDs []string) (Channel, error) {
	conn, err := c.manager.Get(serverID)
	if err != nil {
		return Channel{}, fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{
		"category_id": categoryID,
		"name":        name,
		"type":        typ,
	}
	if len(roleIDs) > 0 {
		payload["role_ids"] = roleIDs
	}

	resp, err := conn.Request(shared.TypeChannelCreate, payload)
	if err != nil {
		return Channel{}, fmt.Errorf("channel.create: %w", err)
	}

	var created struct {
		ID       string `json:"id"`
		Position int    `json:"position"`
		Version  int64  `json:"version"`
	}
	if err := json.Unmarshal(resp.Payload, &created); err != nil {
		return Channel{}, fmt.Errorf("unmarshal channel: %w", err)
	}

	return Channel{
		ID:         created.ID,
		CategoryID: categoryID,
		Name:       name,
		Type:       typ,
		Position:   created.Position,
		RoleIDs:    roleIDs,
	}, nil
}

// UpdateChannel updates a channel on a server.
func (c *ChannelService) UpdateChannel(serverID int64, id string, name string, categoryID string, position int, roleIDs []string) error {
	conn, err := c.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{
		"id":          id,
		"name":        name,
		"category_id": categoryID,
		"position":    position,
	}
	if roleIDs != nil {
		payload["role_ids"] = roleIDs
	}

	_, err = conn.Request(shared.TypeChannelUpdate, payload)
	if err != nil {
		return fmt.Errorf("channel.update: %w", err)
	}
	return nil
}

// DeleteChannel deletes a channel.
func (c *ChannelService) DeleteChannel(serverID int64, id string) error {
	conn, err := c.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeChannelDelete, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return fmt.Errorf("channel.delete: %w", err)
	}
	return nil
}

// channelWS is the wire format from the server.
type channelWS struct {
	ID          string   `json:"id"`
	CategoryID  string   `json:"category_id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Position    int      `json:"position"`
	OpusBitrate int      `json:"opus_bitrate"`
	RoleIDs     []string `json:"role_ids"`
	Version     int64    `json:"version"`
}

func (ch channelWS) toChannel() Channel {
	roleIDs := ch.RoleIDs
	if roleIDs == nil {
		roleIDs = []string{}
	}
	return Channel{
		ID:          ch.ID,
		CategoryID:  ch.CategoryID,
		Name:        ch.Name,
		Type:        ch.Type,
		Position:    ch.Position,
		OpusBitrate: ch.OpusBitrate,
		RoleIDs:     roleIDs,
	}
}
