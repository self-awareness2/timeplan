package schedules

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"chrona/server/internal/db"
)

type Service struct {
	store *db.Store
}

func NewService(store *db.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Export(userID string) (map[string]any, error) {
	items, err := s.loadItems(userID)
	if err != nil {
		return nil, err
	}
	exceptions, err := s.loadExceptions(userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"version":    1,
		"exportedAt": time.Now().UTC().Format(time.RFC3339),
		"schedules":  items,
		"exceptions": exceptions,
	}, nil
}

func (s *Service) Dispatch(userID string, req ActionRequest) (any, error) {
	items, err := s.allItems(userID)
	if err != nil {
		return nil, err
	}
	params := req.Params
	if params == nil {
		params = map[string]any{}
	}

	switch req.Action {
	case "getToday":
		now := todayDate()
		todayItems := itemsOnDay(items, now)
		return map[string]any{
			"date":       now,
			"weekday":    weekdayName(now),
			"pending":    count(todayItems, func(i Item) bool { return i.ExecutionStatus == "not_started" }),
			"inProgress": count(todayItems, func(i Item) bool { return i.ExecutionStatus == "running" }),
			"completed":  count(todayItems, func(i Item) bool { return i.ExecutionStatus == "executed" }),
		}, nil
	case "listDay":
		date := stringParam(params, "date", todayDate())
		return map[string]any{"date": date, "weekday": weekdayName(date), "items": itemsOnDay(items, date)}, nil
	case "listWeek":
		start := startOfWeek(stringParam(params, "date", todayDate()))
		days := make([]map[string]any, 0, 7)
		for i := 0; i < 7; i++ {
			date := addDays(start, i)
			days = append(days, map[string]any{"date": date, "weekday": weekdayName(date), "items": itemsOnDay(items, date)})
		}
		return map[string]any{"weekStart": start, "weekEnd": addDays(start, 6), "weekNumber": 0, "days": days}, nil
	case "listMonth":
		year, month := intParam(params, "year", time.Now().Year()), intParam(params, "month", int(time.Now().Month()))
		totalDays := daysInMonth(year, month)
		days := make([]map[string]any, 0, totalDays)
		monthItems := map[string]Item{}
		for day := 1; day <= totalDays; day++ {
			date := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
			dayItems := itemsOnDay(items, date)
			for _, item := range dayItems {
				monthItems[occurrenceKey(item)] = item
			}
			days = append(days, map[string]any{"date": date, "weekday": weekdayName(date), "count": len(dayItems), "items": dayItems})
		}
		return map[string]any{"year": year, "month": month, "monthName": fmt.Sprintf("%d月", month), "daysInMonth": totalDays, "firstWeekday": dayOfWeek(fmt.Sprintf("%04d-%02d-01", year, month)), "days": days, "items": mapOccurrenceValues(monthItems)}, nil
	case "listYear":
		year := intParam(params, "year", time.Now().Year())
		months := make([]map[string]any, 0, 12)
		total := 0
		for month := 1; month <= 12; month++ {
			monthItems := itemsInMonth(items, year, month)
			total += len(monthItems)
			months = append(months, map[string]any{"month": month, "monthName": fmt.Sprintf("%d月", month), "count": len(monthItems), "items": monthItems})
		}
		yearItems := make([]Item, 0)
		for month := 1; month <= 12; month++ {
			yearItems = append(yearItems, itemsInMonth(items, year, month)...)
		}
		return map[string]any{"year": year, "months": months, "totalCount": total, "items": yearItems}, nil
	case "getItem":
		return s.getItem(userID, int64Param(params, "id", 0))
	case "addItem":
		return s.addItem(userID, draftParam(params))
	case "updateItem":
		return s.updateItem(userID, int64Param(params, "id", 0), draftParam(params), stringParam(params, "scope", "series"), stringParam(params, "occurrenceDate", ""))
	case "deleteItem":
		id := int64Param(params, "id", 0)
		return map[string]any{"id": id}, s.deleteItem(userID, id, stringParam(params, "scope", "series"), stringParam(params, "occurrenceDate", ""))
	case "setExecution":
		return s.setExecution(userID, int64Param(params, "id", 0), stringParam(params, "executionStatus", "not_started"), stringParam(params, "failureReason", ""), stringParam(params, "occurrenceDate", ""))
	case "search":
		keyword := strings.ToLower(stringParam(params, "keyword", ""))
		result := make([]Item, 0)
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.Title+" "+item.Description+" "+item.Category), keyword) {
				result = append(result, item)
			}
		}
		return map[string]any{"keyword": keyword, "items": result}, nil
	case "stats":
		byCategory := map[string]int{}
		byReason := map[string]int{}
		byDate := map[string]map[string]int{}
		historyItems := materializedItems(items)
		for _, item := range historyItems {
			if item.Category != "" {
				byCategory[item.Category]++
			}
			if item.FailureReason != "" {
				byReason[item.FailureReason]++
			}
			if byDate[item.Date] == nil {
				byDate[item.Date] = map[string]int{}
			}
			byDate[item.Date]["planned"] += plannedMinutes(item)
			byDate[item.Date]["actual"] += item.ExecutionMinutes
			if item.ExecutionStatus == "executed" {
				byDate[item.Date]["executed"]++
			}
			if item.ExecutionStatus == "skipped" {
				byDate[item.Date]["skipped"]++
			}
		}
		daily := make([]map[string]any, 0, len(byDate))
		for date, values := range byDate {
			daily = append(daily, map[string]any{"date": date, "plannedMinutes": values["planned"], "actualMinutes": values["actual"], "executed": values["executed"], "skipped": values["skipped"]})
		}
		sort.Slice(daily, func(i, j int) bool { return daily[i]["date"].(string) < daily[j]["date"].(string) })
		return map[string]any{
			"pending":    count(historyItems, func(i Item) bool { return i.ExecutionStatus == "not_started" }),
			"inProgress": count(historyItems, func(i Item) bool { return i.ExecutionStatus == "running" || i.ExecutionStatus == "paused" }),
			"completed":  count(historyItems, func(i Item) bool { return i.ExecutionStatus == "executed" }),
			"cancelled":  count(historyItems, func(i Item) bool { return i.ExecutionStatus == "skipped" || i.ExecutionStatus == "cancelled" }),
			"total":      len(historyItems),
			"execution": map[string]int{
				"notStarted": count(historyItems, func(i Item) bool { return i.ExecutionStatus == "not_started" }),
				"running":    count(historyItems, func(i Item) bool { return i.ExecutionStatus == "running" }),
				"executed":   count(historyItems, func(i Item) bool { return i.ExecutionStatus == "executed" }),
				"delayed":    count(historyItems, func(i Item) bool { return i.ExecutionStatus == "delayed" }),
				"skipped":    count(historyItems, func(i Item) bool { return i.ExecutionStatus == "skipped" }),
			},
			"byCategory": byCategory,
			"byReason":   byReason,
			"daily":      daily,
		}, nil
	case "categories":
		seen := map[string]bool{}
		for _, item := range items {
			if item.Category != "" {
				seen[item.Category] = true
			}
		}
		cats := make([]string, 0, len(seen))
		for cat := range seen {
			cats = append(cats, cat)
		}
		sort.Strings(cats)
		return cats, nil
	default:
		return nil, fmt.Errorf("unknown action: %s", req.Action)
	}
}

func plannedMinutes(item Item) int {
	if item.StartTime == "00:00" || item.EndTime == "00:00" {
		return 0
	}
	start, startErr := time.Parse("15:04", item.StartTime)
	end, endErr := time.Parse("15:04", item.EndTime)
	if startErr != nil || endErr != nil || !end.After(start) {
		return 0
	}
	return int(end.Sub(start).Minutes())
}

func (s *Service) setExecution(userID string, id int64, status, reason, occurrenceDate string) (Item, error) {
	item, err := s.getItem(userID, id)
	if err != nil {
		return Item{}, err
	}
	if occurrenceDate != "" && item.Repeat != "none" && item.SeriesParentID == 0 {
		item, err = s.materializeOccurrence(userID, item, occurrenceDate)
		if err != nil {
			return Item{}, err
		}
	}
	return s.setExecutionForItem(userID, item, status, reason)
}

func (s *Service) setExecutionForItem(userID string, item Item, status, reason string) (Item, error) {
	status = enum(status, "not_started", "not_started", "running", "paused", "executed", "delayed", "skipped", "cancelled")
	now := time.Now().UTC().Format(time.RFC3339)
	actualStart, actualEnd := item.ActualStartAt, item.ActualEndAt
	minutes := item.ExecutionMinutes
	if status == "running" && actualStart == "" {
		actualStart = now
	}
	if (status == "executed" || status == "delayed" || status == "skipped") && actualEnd == "" {
		actualEnd = now
	}
	if actualStart != "" && actualEnd != "" {
		start, startErr := time.Parse(time.RFC3339, actualStart)
		end, endErr := time.Parse(time.RFC3339, actualEnd)
		if startErr == nil && endErr == nil && end.After(start) {
			minutes = int(end.Sub(start).Minutes())
		}
	}
	_, err := s.store.DB.Exec(`UPDATE schedules SET execution_status = ?, actual_start_at = ?, actual_end_at = ?, execution_minutes = ?, failure_reason = ?, updated_at = ? WHERE user_id = ? AND id = ?`, status, actualStart, actualEnd, minutes, strings.TrimSpace(reason), now, userID, item.ID)
	if err != nil {
		return Item{}, err
	}
	return s.getItem(userID, item.ID)
}

func (s *Service) allItems(userID string) ([]Item, error) {
	items, err := s.loadItems(userID)
	if err != nil {
		return nil, err
	}

	today := todayDate()
	changed := false
	for _, item := range items {
		if item.ExecutionStatus != "not_started" {
			continue
		}
		if item.Repeat == "none" {
			if dueToStartOn(item, today) {
				if _, err := s.setExecutionForItem(userID, item, "running", ""); err != nil {
					return nil, err
				}
				changed = true
			}
			continue
		}
		if item.SeriesParentID == 0 && occursOn(item, today) && dueToStartOn(item, today) {
			override, err := s.materializeOccurrence(userID, item, today)
			if err != nil {
				return nil, err
			}
			if _, err := s.setExecutionForItem(userID, override, "running", ""); err != nil {
				return nil, err
			}
			changed = true
		}
	}
	if changed {
		return s.loadItems(userID)
	}
	return items, nil
}

func (s *Service) loadItems(userID string) ([]Item, error) {
	rows, err := s.store.DB.Query(`SELECT id, title, description, date, start_time, end_time, repeat_type, priority, status, execution_status, actual_start_at, actual_end_at, execution_minutes, failure_reason, category, series_parent_id, created_at, updated_at FROM schedules WHERE user_id = ? ORDER BY date, start_time, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Item, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	exceptions, err := s.loadExceptions(userID)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index].Exceptions = exceptions[items[index].ID]
	}
	return items, nil
}

func dueToStartOn(item Item, date string) bool {
	if date != todayDate() || item.StartTime == "00:00" {
		return false
	}
	now := time.Now().Format("15:04")
	return item.StartTime <= now
}

func (s *Service) getItem(userID string, id int64) (Item, error) {
	row := s.store.DB.QueryRow(`SELECT id, title, description, date, start_time, end_time, repeat_type, priority, status, execution_status, actual_start_at, actual_end_at, execution_minutes, failure_reason, category, series_parent_id, created_at, updated_at FROM schedules WHERE user_id = ? AND id = ?`, userID, id)
	return scanItem(row)
}

func (s *Service) addItem(userID string, draft Draft) (any, error) {
	item := cleanDraft(draft)
	result, err := s.store.DB.Exec(`INSERT INTO schedules (user_id, title, description, date, start_time, end_time, repeat_type, priority, status, execution_status, failure_reason, category, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, userID, item.Title, item.Description, item.Date, item.StartTime, item.EndTime, item.Repeat, item.Priority, item.Status, item.ExecutionStatus, item.FailureReason, item.Category, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	created, err := s.getItem(userID, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": created.ID, "item": created}, nil
}

func (s *Service) updateItem(userID string, id int64, draft Draft, scope, occurrenceDate string) (Item, error) {
	existing, err := s.getItem(userID, id)
	if err != nil {
		return Item{}, err
	}
	root, err := s.seriesRoot(userID, existing)
	if err != nil {
		return Item{}, err
	}
	scope = enum(scope, "series", "this", "series")
	if scope == "series" && root.Repeat != "none" {
		return s.updateStoredItem(userID, root.ID, draft)
	}
	if scope == "this" && root.Repeat != "none" {
		date := fallback(occurrenceDate, existing.Date)
		draft.Repeat = "none"
		if existing.SeriesParentID > 0 {
			return s.updateStoredItem(userID, existing.ID, draft)
		}
		override, err := s.materializeOccurrence(userID, root, date)
		if err != nil {
			return Item{}, err
		}
		return s.updateStoredItem(userID, override.ID, draft)
	}
	return s.updateStoredItem(userID, existing.ID, draft)
}

func (s *Service) updateStoredItem(userID string, id int64, draft Draft) (Item, error) {
	item := cleanDraft(draft)
	_, err := s.store.DB.Exec(`UPDATE schedules SET title = ?, description = ?, date = ?, start_time = ?, end_time = ?, repeat_type = ?, priority = ?, status = ?, execution_status = ?, failure_reason = ?, category = ?, updated_at = ? WHERE user_id = ? AND id = ?`, item.Title, item.Description, item.Date, item.StartTime, item.EndTime, item.Repeat, item.Priority, item.Status, item.ExecutionStatus, item.FailureReason, item.Category, item.UpdatedAt, userID, id)
	if err != nil {
		return Item{}, err
	}
	return s.getItem(userID, id)
}

func (s *Service) deleteItem(userID string, id int64, scope, occurrenceDate string) error {
	existing, err := s.getItem(userID, id)
	if err != nil {
		return err
	}
	root, err := s.seriesRoot(userID, existing)
	if err != nil {
		return err
	}
	scope = enum(scope, "series", "this", "series")
	if scope == "series" && root.Repeat != "none" {
		if _, err := s.store.DB.Exec(`DELETE FROM schedule_exceptions WHERE user_id = ? AND schedule_id = ?`, userID, root.ID); err != nil {
			return err
		}
		_, err := s.store.DB.Exec(`DELETE FROM schedules WHERE user_id = ? AND (id = ? OR series_parent_id = ?)`, userID, root.ID, root.ID)
		return err
	}
	if scope == "this" && root.Repeat != "none" {
		date := fallback(occurrenceDate, existing.Date)
		if existing.SeriesParentID > 0 {
			if _, err := s.store.DB.Exec(`DELETE FROM schedules WHERE user_id = ? AND id = ?`, userID, existing.ID); err != nil {
				return err
			}
		}
		return s.excludeOccurrence(userID, root.ID, date)
	}
	result, err := s.store.DB.Exec(`DELETE FROM schedules WHERE user_id = ? AND id = ?`, userID, id)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return errors.New("item not found")
	}
	return nil
}

func (s *Service) seriesRoot(userID string, item Item) (Item, error) {
	if item.SeriesParentID == 0 {
		return item, nil
	}
	return s.getItem(userID, item.SeriesParentID)
}

func (s *Service) loadExceptions(userID string) (map[int64]map[string]bool, error) {
	rows, err := s.store.DB.Query(`SELECT schedule_id, occurrence_date FROM schedule_exceptions WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	exceptions := make(map[int64]map[string]bool)
	for rows.Next() {
		var scheduleID int64
		var occurrenceDate string
		if err := rows.Scan(&scheduleID, &occurrenceDate); err != nil {
			return nil, err
		}
		if exceptions[scheduleID] == nil {
			exceptions[scheduleID] = make(map[string]bool)
		}
		exceptions[scheduleID][occurrenceDate] = true
	}
	return exceptions, rows.Err()
}

func (s *Service) excludeOccurrence(userID string, scheduleID int64, occurrenceDate string) error {
	_, err := s.store.DB.Exec(`INSERT OR IGNORE INTO schedule_exceptions (user_id, schedule_id, occurrence_date, created_at) VALUES (?, ?, ?, ?)`, userID, scheduleID, occurrenceDate, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Service) materializeOccurrence(userID string, parent Item, occurrenceDate string) (Item, error) {
	root, err := s.seriesRoot(userID, parent)
	if err != nil {
		return Item{}, err
	}
	row := s.store.DB.QueryRow(`SELECT id, title, description, date, start_time, end_time, repeat_type, priority, status, execution_status, actual_start_at, actual_end_at, execution_minutes, failure_reason, category, series_parent_id, created_at, updated_at FROM schedules WHERE user_id = ? AND series_parent_id = ? AND date = ?`, userID, root.ID, occurrenceDate)
	existing, err := scanItem(row)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Item{}, err
	}
	if err := s.excludeOccurrence(userID, root.ID, occurrenceDate); err != nil {
		return Item{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.store.DB.Exec(`INSERT INTO schedules (user_id, title, description, date, start_time, end_time, repeat_type, priority, status, execution_status, actual_start_at, actual_end_at, execution_minutes, failure_reason, category, series_parent_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 'none', ?, ?, 'not_started', '', '', 0, '', ?, ?, ?, ?)`, userID, root.Title, root.Description, occurrenceDate, root.StartTime, root.EndTime, root.Priority, root.Status, root.Category, root.ID, now, now)
	if err != nil {
		return Item{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Item{}, err
	}
	return s.getItem(userID, id)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanItem(row scanner) (Item, error) {
	var item Item
	err := row.Scan(&item.ID, &item.Title, &item.Description, &item.Date, &item.StartTime, &item.EndTime, &item.Repeat, &item.Priority, &item.Status, &item.ExecutionStatus, &item.ActualStartAt, &item.ActualEndAt, &item.ExecutionMinutes, &item.FailureReason, &item.Category, &item.SeriesParentID, &item.CreatedAt, &item.UpdatedAt)
	item.HasTime = item.StartTime != "00:00" || item.EndTime != "00:00"
	return item, err
}

func cleanDraft(d Draft) Item {
	now := todayDate()
	item := Item{
		Title:           strings.TrimSpace(d.Title),
		Description:     d.Description,
		Date:            fallback(d.Date, now),
		StartTime:       fallback(d.StartTime, "00:00"),
		EndTime:         fallback(d.EndTime, "00:00"),
		Repeat:          enum(d.Repeat, "none", "daily", "weekly", "monthly", "yearly"),
		Priority:        enum(d.Priority, "medium", "low", "medium", "high"),
		Status:          enum(d.Status, "pending", "pending", "in_progress", "completed", "cancelled"),
		ExecutionStatus: enum(d.ExecutionStatus, "not_started", "not_started", "running", "paused", "executed", "delayed", "skipped", "cancelled"),
		FailureReason:   strings.TrimSpace(d.FailureReason),
		Category:        d.Category,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	item.HasTime = item.StartTime != "00:00" || item.EndTime != "00:00"
	return item
}

func todayDate() string { return time.Now().Format("2006-01-02") }

func dayOfWeek(date string) int {
	t, _ := time.Parse("2006-01-02", date)
	return int(t.Weekday())
}

func weekdayName(date string) string {
	return []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}[dayOfWeek(date)]
}

func addDays(date string, delta int) string {
	t, _ := time.Parse("2006-01-02", date)
	return t.AddDate(0, 0, delta).Format("2006-01-02")
}

func startOfWeek(date string) string {
	dow := dayOfWeek(date)
	if dow == 0 {
		return addDays(date, -6)
	}
	return addDays(date, 1-dow)
}

func daysInMonth(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func occursOn(item Item, date string) bool {
	if item.Status == "cancelled" || date < item.Date {
		return false
	}
	if item.Exceptions[date] {
		return false
	}
	switch item.Repeat {
	case "daily":
		return true
	case "weekly":
		return dayOfWeek(item.Date) == dayOfWeek(date)
	case "monthly":
		return item.Date[8:10] == date[8:10]
	case "yearly":
		return item.Date[5:] == date[5:]
	default:
		return item.Date == date
	}
}

func itemsOnDay(items []Item, date string) []Item {
	result := make([]Item, 0)
	for _, item := range items {
		if occursOn(item, date) {
			occurrence := item
			occurrence.OccurrenceDate = date
			if item.Repeat != "none" && item.Date != date {
				occurrence.Date = date
				occurrence.IsVirtual = true
			}
			result = append(result, occurrence)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime < result[j].StartTime
	})
	return result
}

func itemsInMonth(items []Item, year, month int) []Item {
	values := make(map[string]Item)
	for day := 1; day <= daysInMonth(year, month); day++ {
		date := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		for _, item := range itemsOnDay(items, date) {
			values[occurrenceKey(item)] = item
		}
	}
	return mapOccurrenceValues(values)
}

func occurrenceKey(item Item) string {
	return fmt.Sprintf("%d:%s", item.ID, fallback(item.OccurrenceDate, item.Date))
}

func count(items []Item, fn func(Item) bool) int {
	total := 0
	for _, item := range items {
		if fn(item) {
			total++
		}
	}
	return total
}

func materializedItems(items []Item) []Item {
	result := make([]Item, 0, len(items))
	for _, item := range items {
		if item.Repeat != "none" && item.SeriesParentID == 0 {
			continue
		}
		result = append(result, item)
	}
	return result
}

func stringParam(params map[string]any, key, fallbackValue string) string {
	if value, ok := params[key].(string); ok && value != "" {
		return value
	}
	return fallbackValue
}

func intParam(params map[string]any, key string, fallbackValue int) int {
	switch value := params[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case string:
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallbackValue
}

func int64Param(params map[string]any, key string, fallbackValue int64) int64 {
	return int64(intParam(params, key, int(fallbackValue)))
}

func draftParam(params map[string]any) Draft {
	raw, _ := params["item"].(map[string]any)
	return Draft{
		Title:           stringFromMap(raw, "title"),
		Description:     stringFromMap(raw, "description"),
		Date:            stringFromMap(raw, "date"),
		StartTime:       stringFromMap(raw, "startTime"),
		EndTime:         stringFromMap(raw, "endTime"),
		Repeat:          stringFromMap(raw, "repeat"),
		Priority:        stringFromMap(raw, "priority"),
		Status:          stringFromMap(raw, "status"),
		ExecutionStatus: stringFromMap(raw, "executionStatus"),
		FailureReason:   stringFromMap(raw, "failureReason"),
		Category:        stringFromMap(raw, "category"),
	}
}

func stringFromMap(values map[string]any, key string) string {
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func fallback(value, fallbackValue string) string {
	if value == "" {
		return fallbackValue
	}
	return value
}

func enum(value, fallbackValue string, allowed ...string) string {
	for _, item := range allowed {
		if value == item {
			return value
		}
	}
	return fallbackValue
}

func mapValues(values map[int64]Item) []Item {
	result := make([]Item, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date+" "+result[i].StartTime < result[j].Date+" "+result[j].StartTime
	})
	return result
}

func mapOccurrenceValues(values map[string]Item) []Item {
	result := make([]Item, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date+" "+result[i].StartTime < result[j].Date+" "+result[j].StartTime
	})
	return result
}
