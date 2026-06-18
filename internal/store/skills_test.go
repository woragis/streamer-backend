package store_test

import (
	"context"
	"testing"

	"github.com/woragis/streamer-backend/internal/calisthenics"
	"github.com/woragis/streamer-backend/internal/db/testutil"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/leetcode"
	"github.com/woragis/streamer-backend/internal/store"
)

func TestSkillCatalogAndAcquisition(t *testing.T) {
	sqlDB := testutil.Open(t)

	st := store.New(sqlDB)
	ctx := context.Background()
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}

	movements, err := st.ListMovements(ctx, defaults.DefaultRoomID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(movements) != 5 {
		t.Fatalf("expected 5 seed movements, got %d", len(movements))
	}

	sess, err := st.CreateLiveSession(ctx, defaults.DefaultRoomID, leetcode.CreateLiveSessionInput{
		Domain: "calisthenics", Platforms: []string{"youtube"},
	})
	if err != nil {
		t.Fatal(err)
	}

	acq, err := st.CreateAcquisition(ctx, defaults.DefaultRoomID, calisthenics.CreateAcquisitionInput{
		MovementID:       defaults.ScopedSeedID(defaults.DefaultRoomID, "muscle-up"),
		LiveSessionID:    &sess.ID,
		ProficiencyAfter: calisthenics.ProficiencyConsistent,
		Notes:            "Primeiro rep limpo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if acq.ProficiencyBefore != calisthenics.ProficiencyUnknown {
		t.Fatalf("expected before unknown, got %s", acq.ProficiencyBefore)
	}

	state, err := st.GetCalisthenicsState(ctx, defaults.DefaultRoomID)
	if err != nil {
		t.Fatal(err)
	}
	if state.SkillAlert == nil {
		t.Fatal("expected skillAlert in calisthenics state")
	}
	if state.SkillAlert.MovementName != "Muscle Up" {
		t.Fatalf("unexpected movement name %q", state.SkillAlert.MovementName)
	}

	stats, err := st.GetSkillStats(ctx, defaults.DefaultRoomID, "", sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stats.AcquisitionCount != 1 {
		t.Fatalf("expected 1 acquisition in session, got %d", stats.AcquisitionCount)
	}

	if err := st.AcknowledgeAcquisition(ctx, defaults.DefaultRoomID, acq.ID); err != nil {
		t.Fatal(err)
	}
	state2, err := st.GetCalisthenicsState(ctx, defaults.DefaultRoomID)
	if err != nil {
		t.Fatal(err)
	}
	if state2.SkillAlert != nil {
		t.Fatal("expected skillAlert cleared after ack")
	}

	prof, err := st.GetProficiency(ctx, defaults.DefaultRoomID, defaults.ScopedSeedID(defaults.DefaultRoomID, "muscle-up"))
	if err != nil {
		t.Fatal(err)
	}
	if prof.Level != calisthenics.ProficiencyConsistent {
		t.Fatalf("expected consistent proficiency, got %s", prof.Level)
	}
}

func TestListAcquisitionsByMonth(t *testing.T) {
	sqlDB := testutil.Open(t)

	st := store.New(sqlDB)
	ctx := context.Background()
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := st.CreateAcquisition(ctx, defaults.DefaultRoomID, calisthenics.CreateAcquisitionInput{
		MovementID: "handstand", ProficiencyAfter: calisthenics.ProficiencyLearning,
	})
	if err != nil {
		t.Fatal(err)
	}

	items, err := st.ListAcquisitions(ctx, defaults.DefaultRoomID, "2026-06", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 acquisition in month, got %d", len(items))
	}
}
