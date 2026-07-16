package schedules

import (
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
		return map[string]any{
			"date":       now,
			"weekday":    weekdayName(now),
			"pending":    count(items, func(i Item) bool { return i.Status == "pending" || i.Status == "in_progress" }),
			"inProgress": count(items, func(i Item) bool { return i.Status == "in_progress" }),
			"completed":  count(items, func(i Item) bool { return i.Status == "completed" }),
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
		monthItems := map[int64]Item{}
		for day := 1; day <= totalDays; day++ {
			date := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
			dayItems := itemsOnDay(items, date)
			for _, item := range dayItems {
				monthItems[item.ID] = item
			}
			days = append(days, map[string]any{"date": date, "weekday": weekdayName(date), "count": len(dayItems), "items": dayItems})
		}
		return map[string]any{"year": year, "month": month, "monthName": fmt.Sprintf("%d月", month), "daysInMonth": totalDays, "firstWeekday": dayOfWeek(fmt.Sprintf("%04d-%02d-01", year, month)), "days": days, "items": mapValues(monthItems)}, nil
	case "listYear":
		year := intParam(params, "year", time.Now().Year())
		months := make([]map[string]any, 0, 12)
		total := 0
		for month := 1; month <= 12; month++ {
			prefix := fmt.Sprintf("%04d-%02d", year, month)
			monthItems := make([]Item, 0)
			for _, item := range items {
				if strings.HasPrefix(item.Date, prefix) {
					monthItems = append(monthItems, item)
				}
			}
			total += len(monthItems)
			months = append(months, map[string]any{"month": month, "monthName": fmt.Sprintf("%d月", month), "count": len(monthItems), "items": monthItems})
		}
		yearItems := make([]Item, 0)
		for _, item := range items {
			if strings.HasPrefix(item.Date, fmt.Sprintf("%04d-", year)) {
				yearItems = append(yearItems, item)
			}
		}
		return map[string]any{"year": year, "months": months, "totalCount": total, "items": yearItems}, nil
	case "getItem":
		return s.getItem(userID, int64Param(params, "id", 0))
	case "addItem":
		return s.addItem(userID, draftParam(params))
	case "updateItem":
		return s.updateItem(userID, int64Param(params, "id", 0), draftParam(params))
	case "deleteItem":
		id := int64Param(params, "id", 0)
		return map[string]any{"id": id}, s.deleteItem(userID, id)
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
		for _, item := range items {
			if item.Category != "" {
				byCategory[item.Category]++
			}
		}
		return map[string]any{
			"pending":    count(items, func(i Item) bool { return i.Status == "pending" || i.Status == "in_progress" }),
			"inProgress": count(items, func(i Item) bool { return i.Status == "in_progress" }),
			"completed":  count(items, func(i Item) bool { return i.Status == "completed" }),
			"cancelled":  count(items, func(i Item) bool { return i.Status == "cancelled" }),
			"total":      len(items),
			"byCategory": byCategory,
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

func (s *Service) allItems(userID string) ([]Item, error) {
	rows, err := s.store.DB.Query(`SELECT id, title, description, date, start_time, end_time, repeat_type, priority, status, category, created_at, updated_at FROM schedules WHERE user_id = ? ORDER BY date, start_time, id`, userID)
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
	return items, rows.Err()
}

func (s *Service) getItem(userID string, id int64) (Item, error) {
	row := s.store.DB.QueryRow(`SELECT id, title, description, date, start_time, end_time, repeat_type, priority, status, category, created_at, updated_at FROM schedules WHERE user_id = ? AND id = ?`, userID, id)
	return scanItem(row)
}

func (s *Service) addItem(userID string, draft Draft) (any, error) {
	item := cleanDraft(draft)
	result, err := s.store.DB.Exec(`INSERT INTO schedules (user_id, title, description, date, start_time, end_time, repeat_type, priority, status, category, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, userID, item.Title, item.Description, item.Date, item.StartTime, item.EndTime, item.Repeat, item.Priority, item.Status, item.Category, item.CreatedAt, item.UpdatedAt)
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

func (s *Service) updateItem(userID string, id int64, draft Draft) (Item, error) {
	if _, err := s.getItem(userID, id); err != nil {
		return Item{}, err
	}
	item := cleanDraft(draft)
	_, err := s.store.DB.Exec(`UPDATE schedules SET title = ?, description = ?, date = ?, start_time = ?, end_time = ?, repeat_type = ?, priority = ?, status = ?, category = ?, updated_at = ? WHERE user_id = ? AND id = ?`, item.Title, item.Description, item.Date, item.StartTime, item.EndTime, item.Repeat, item.Priority, item.Status, item.Category, item.UpdatedAt, userID, id)
	if err != nil {
		return Item{}, err
	}
	return s.getItem(userID, id)
}

func (s *Service) deleteItem(userID string, id int64) error {
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

type scanner interface {
	Scan(dest ...any) error
}

func scanItem(row scanner) (Item, error) {
	var item Item
	err := row.Scan(&item.ID, &item.Title, &item.Description, &item.Date, &item.StartTime, &item.EndTime, &item.Repeat, &item.Priority, &item.Status, &item.Category, &item.CreatedAt, &item.UpdatedAt)
	item.HasTime = item.StartTime != "00:00" || item.EndTime != "00:00"
	return item, err
}

func cleanDraft(d Draft) Item {
	now := todayDate()
	item := Item{
		Title:       strings.TrimSpace(d.Title),
		Description: d.Description,
		Date:        fallback(d.Date, now),
		StartTime:   fallback(d.StartTime, "00:00"),
		EndTime:     fallback(d.EndTime, "00:00"),
		Repeat:      enum(d.Repeat, "none", "daily", "weekly", "monthly", "yearly"),
		Priority:    enum(d.Priority, "medium", "low", "medium", "high"),
		Status:      enum(d.Status, "pending", "pending", "in_progress", "completed", "cancelled"),
		Category:    d.Category,
		CreatedAt:   now,
		UpdatedAt:   now,
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
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime < result[j].StartTime
	})
	return result
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
		Title:       stringFromMap(raw, "title"),
		Description: stringFromMap(raw, "description"),
		Date:        stringFromMap(raw, "date"),
		StartTime:   stringFromMap(raw, "startTime"),
		EndTime:     stringFromMap(raw, "endTime"),
		Repeat:      stringFromMap(raw, "repeat"),
		Priority:    stringFromMap(raw, "priority"),
		Status:      stringFromMap(raw, "status"),
		Category:    stringFromMap(raw, "category"),
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
