package templates

import (
	"testing"

	"github.com/streambinder/foedus/internal/models"
)

func TestGiftsHaveInvitationIDs(t *testing.T) {
	if giftsHaveInvitationIDs(nil) {
		t.Fatal("expected false for no gifts")
	}
	if giftsHaveInvitationIDs([]models.Gift{{Donor: "x"}}) {
		t.Fatal("expected false when no gift has an invitation")
	}
	id := 3
	if !giftsHaveInvitationIDs([]models.Gift{{Donor: "x", InvitationID: &id}}) {
		t.Fatal("expected true when a gift has an invitation")
	}
}

func TestGiftInvitationGuestFirstNames(t *testing.T) {
	invitations := []models.Invitation{
		{ID: 3, Guests: []models.Guest{{FirstName: "Agnese"}, {FirstName: "Davide"}}},
	}
	if got := giftInvitationGuestFirstNames(invitations, nil); got != "" {
		t.Fatalf("expected empty for nil invitation id, got %q", got)
	}
	id := 3
	if got := giftInvitationGuestFirstNames(invitations, &id); got != "Agnese, Davide" {
		t.Fatalf("expected joined first names, got %q", got)
	}
	unknown := 99
	if got := giftInvitationGuestFirstNames(invitations, &unknown); got != "" {
		t.Fatalf("expected empty for unknown invitation id, got %q", got)
	}
}
