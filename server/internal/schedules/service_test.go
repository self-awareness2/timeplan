package schedules

import (
	"path/filepath"
	"testing"

	"chrona/server/internal/db"
)

const testUserID = "test-user"

func TestRecurringOccurrenceCanBeEditedIndependently(t *testing.T) {
	service := newTestService(t)
	root := addRecurringItem(t, service, "Weekly review")

	occurrenceDate := "2026-07-13"
	virtual := occurrenceOn(t, service, occurrenceDate)
	if !virtual.IsVirtual || virtual.ID != root.ID || virtual.OccurrenceDate != occurrenceDate {
		t.Fatalf("expected a virtual weekly occurrence, got %#v", virtual)
	}

	override, err := service.updateItem(testUserID, root.ID, Draft{
		Title: "Rescheduled review", Date: occurrenceDate, StartTime: "00:00", EndTime: "00:00",
		Repeat: "weekly", Priority: "medium", Status: "pending",
	}, "this", occurrenceDate)
	if err != nil {
		t.Fatalf("update single occurrence: %v", err)
	}
	if override.SeriesParentID != root.ID || override.Repeat != "none" {
		t.Fatalf("expected a one-off override, got %#v", override)
	}

	updated := occurrenceOn(t, service, occurrenceDate)
	if updated.Title != "Rescheduled review" || updated.IsVirtual {
		t.Fatalf("expected the override on the selected date, got %#v", updated)
	}
	future := occurrenceOn(t, service, "2026-07-20")
	if future.Title != "Weekly review" || !future.IsVirtual {
		t.Fatalf("expected future series instances to remain unchanged, got %#v", future)
	}
}

func TestRecurringOccurrenceCanBeDeletedWithoutDeletingSeries(t *testing.T) {
	service := newTestService(t)
	root := addRecurringItem(t, service, "Weekly review")

	deletedDate := "2026-07-13"
	if err := service.deleteItem(testUserID, root.ID, "this", deletedDate); err != nil {
		t.Fatalf("delete single occurrence: %v", err)
	}

	items, err := service.allItems(testUserID)
	if err != nil {
		t.Fatal(err)
	}
	if got := itemsOnDay(items, deletedDate); len(got) != 0 {
		t.Fatalf("expected the deleted occurrence to be excluded, got %#v", got)
	}
	if later := occurrenceOn(t, service, "2026-07-20"); later.ID != root.ID || !later.IsVirtual {
		t.Fatalf("expected the rest of the series to remain, got %#v", later)
	}
}

func TestSeriesUpdateChangesFutureOccurrences(t *testing.T) {
	service := newTestService(t)
	root := addRecurringItem(t, service, "Weekly review")

	_, err := service.updateItem(testUserID, root.ID, Draft{
		Title: "Updated weekly review", Date: root.Date, StartTime: "00:00", EndTime: "00:00",
		Repeat: "weekly", Priority: "high", Status: "pending",
	}, "series", root.Date)
	if err != nil {
		t.Fatalf("update series: %v", err)
	}
	if got := occurrenceOn(t, service, "2026-07-20"); got.Title != "Updated weekly review" || got.Priority != "high" {
		t.Fatalf("expected future occurrence to use series update, got %#v", got)
	}
}

func TestExportIncludesRecurringExceptions(t *testing.T) {
	service := newTestService(t)
	root := addRecurringItem(t, service, "Weekly review")
	if err := service.deleteItem(testUserID, root.ID, "this", "2026-07-13"); err != nil {
		t.Fatal(err)
	}

	exported, err := service.Export(testUserID)
	if err != nil {
		t.Fatal(err)
	}
	exceptions := exported["exceptions"].(map[int64]map[string]bool)
	if !exceptions[root.ID]["2026-07-13"] {
		t.Fatalf("expected export to include the occurrence exception: %#v", exceptions)
	}
}

func TestExecutionMaterializesRecurringOccurrenceForStats(t *testing.T) {
	service := newTestService(t)
	root := addRecurringItem(t, service, "Weekly review")

	completed, err := service.setExecution(testUserID, root.ID, "executed", "", "2026-07-13")
	if err != nil {
		t.Fatal(err)
	}
	if completed.SeriesParentID != root.ID || completed.ExecutionStatus != "executed" {
		t.Fatalf("expected completed occurrence record, got %#v", completed)
	}

	stats, err := service.Dispatch(testUserID, ActionRequest{Action: "stats"})
	if err != nil {
		t.Fatal(err)
	}
	values := stats.(map[string]any)
	if values["completed"] != 1 || values["total"] != 1 {
		t.Fatalf("expected one materialized execution in stats, got %#v", values)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "chrona.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.DB.Exec(`INSERT INTO users (id, username, password_hash, created_at) VALUES (?, 'tester', 'hash', '2026-01-01T00:00:00Z')`, testUserID); err != nil {
		t.Fatal(err)
	}
	return NewService(store)
}

func addRecurringItem(t *testing.T, service *Service, title string) Item {
	t.Helper()
	created, err := service.addItem(testUserID, Draft{
		Title: title, Date: "2026-07-06", StartTime: "00:00", EndTime: "00:00",
		Repeat: "weekly", Priority: "medium", Status: "pending",
	})
	if err != nil {
		t.Fatal(err)
	}
	return created.(map[string]any)["item"].(Item)
}

func occurrenceOn(t *testing.T, service *Service, date string) Item {
	t.Helper()
	items, err := service.allItems(testUserID)
	if err != nil {
		t.Fatal(err)
	}
	onDate := itemsOnDay(items, date)
	if len(onDate) != 1 {
		t.Fatalf("expected one item on %s, got %#v", date, onDate)
	}
	return onDate[0]
}
