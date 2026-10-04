package services

import (
	"broadcast-tool/models"
	"testing"
)

func TestSlotIdentityNotPosition(t *testing.T) {
	old := []models.TimeSlot{{ID: "early", DayIndex: 1, Order: 1, Time: "06:20"}, {ID: "late", DayIndex: 1, Order: 2, Time: "18:30"}}
	t.Run("insertion cannot steal later explicit ID", func(t *testing.T) {
		got := reconcileSlots(old, []models.TimeSlot{{ID: "new", DayIndex: 1, Order: 1, Time: "12:00"}, {ID: "early", DayIndex: 1, Order: 2, Time: "06:10"}, {ID: "late", DayIndex: 1, Order: 3, Time: "18:30"}})
		if got[0].ID != "new" || got[1].ID != "early" || got[2].ID != "late" {
			t.Fatal(got)
		}
	})
	t.Run("copied reordered rows match clock time", func(t *testing.T) {
		got := reconcileSlots(old, []models.TimeSlot{{ID: "copy1", DayIndex: 1, Order: 1, Time: "18:30"}, {ID: "copy2", DayIndex: 1, Order: 2, Time: "06:20"}})
		if got[0].ID != "late" || got[1].ID != "early" {
			t.Fatal(got)
		}
	})
	t.Run("different time with new ID needs deletion confirmation", func(t *testing.T) {
		got := reconcileSlots(old, []models.TimeSlot{{ID: "new", DayIndex: 1, Order: 1, Time: "12:00"}})
		if got[0].ID != "new" {
			t.Fatal(got)
		}
	})
}

func TestInvalidSlotIdentitiesAreRejected(t *testing.T) {
	old := []models.TimeSlot{{ID: "original", DayIndex: 1, Order: 1, Time: "06:20"}}
	for _, slots := range [][]models.TimeSlot{
		{{ID: "original", DayIndex: 2, Order: 1, Time: "06:20"}},
		{{ID: "original", DayIndex: 1, Order: 1, Time: "06:20"}, {ID: "original", DayIndex: 1, Order: 2, Time: "18:30"}},
		{{ID: "new", DayIndex: 1, Order: 1, Time: "26:00"}},
	} {
		if err := ValidateTimeSlots(old, slots); err == nil {
			t.Fatal("invalid identity accepted", slots)
		}
	}
}
