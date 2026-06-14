package calisthenics

// DefaultSkillCatalog returns seed categories and movements for a new room.
func DefaultSkillCatalog(roomID string) ([]MovementCategory, []Movement) {
	categories := []MovementCategory{
		{ID: "pull", RoomID: roomID, Name: "Pull"},
		{ID: "push", RoomID: roomID, Name: "Push"},
		{ID: "core", RoomID: roomID, Name: "Core"},
		{ID: "skill", RoomID: roomID, Name: "Skill"},
		{ID: "hold", RoomID: roomID, Name: "Hold"},
	}
	movements := []Movement{
		{ID: "muscle-up", RoomID: roomID, Slug: "muscle-up", Name: "Muscle Up", CategoryID: "pull", Description: "Explosive pull to support above the bar"},
		{ID: "front-lever", RoomID: roomID, Slug: "front-lever", Name: "Front Lever", CategoryID: "hold", Description: "Horizontal hold facing up"},
		{ID: "handstand", RoomID: roomID, Slug: "handstand", Name: "Handstand", CategoryID: "skill", Description: "Freestanding handstand balance"},
		{ID: "l-sit", RoomID: roomID, Slug: "l-sit", Name: "L-Sit", CategoryID: "core", Description: "Legs extended parallel to ground"},
		{ID: "planche", RoomID: roomID, Slug: "planche", Name: "Planche", CategoryID: "skill", Description: "Horizontal body hold on hands"},
	}
	return categories, movements
}
