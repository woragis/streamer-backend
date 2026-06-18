package calisthenics

import "github.com/woragis/streamer-backend/internal/defaults"

// DefaultSkillCatalog returns seed categories and movements for a new room.
func DefaultSkillCatalog(roomID string) ([]MovementCategory, []Movement) {
	scope := defaults.ScopedSeedID
	categories := []MovementCategory{
		{ID: scope(roomID, "pull"), RoomID: roomID, Name: "Pull"},
		{ID: scope(roomID, "push"), RoomID: roomID, Name: "Push"},
		{ID: scope(roomID, "core"), RoomID: roomID, Name: "Core"},
		{ID: scope(roomID, "skill"), RoomID: roomID, Name: "Skill"},
		{ID: scope(roomID, "hold"), RoomID: roomID, Name: "Hold"},
	}
	movements := []Movement{
		{ID: scope(roomID, "muscle-up"), RoomID: roomID, Slug: "muscle-up", Name: "Muscle Up", CategoryID: scope(roomID, "pull"), Description: "Explosive pull to support above the bar"},
		{ID: scope(roomID, "front-lever"), RoomID: roomID, Slug: "front-lever", Name: "Front Lever", CategoryID: scope(roomID, "hold"), Description: "Horizontal hold facing up"},
		{ID: scope(roomID, "handstand"), RoomID: roomID, Slug: "handstand", Name: "Handstand", CategoryID: scope(roomID, "skill"), Description: "Freestanding handstand balance"},
		{ID: scope(roomID, "l-sit"), RoomID: roomID, Slug: "l-sit", Name: "L-Sit", CategoryID: scope(roomID, "core"), Description: "Legs extended parallel to ground"},
		{ID: scope(roomID, "planche"), RoomID: roomID, Slug: "planche", Name: "Planche", CategoryID: scope(roomID, "skill"), Description: "Horizontal body hold on hands"},
	}
	return categories, movements
}
