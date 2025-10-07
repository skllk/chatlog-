package v4

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/sjzar/chatlog/internal/model"
)

func fillContactLabelsV4(ctx context.Context, db *sql.DB, contacts []*model.ContactV4) error {
	if len(contacts) == 0 {
		return nil
	}

	labelMap, err := loadAllLabels(ctx, db)
	if err != nil {
		if isNoSuchTableErr(err) {
			labelMap = map[int]string{}
		} else {
			return fmt.Errorf("load labels: %w", err)
		}
	}

	link := map[string][]string{}

	hasListColumn, err := columnExists(ctx, db, "contact", "LabelIDList")
	if err != nil && !isNoSuchTableErr(err) {
		return fmt.Errorf("check LabelIDList column: %w", err)
	}

	if hasListColumn {
		rows, err := db.QueryContext(ctx, `SELECT username, LabelIDList FROM contact`)
		if err != nil {
			if !isNoSuchTableErr(err) {
				return fmt.Errorf("query label id list: %w", err)
			}
		} else {
			defer rows.Close()
			for rows.Next() {
				var username string
				var raw sql.NullString
				if err := rows.Scan(&username, &raw); err != nil {
					return fmt.Errorf("scan label id list: %w", err)
				}
				if !raw.Valid || strings.TrimSpace(raw.String) == "" {
					continue
				}
				ids := splitIDs(raw.String)
				for _, id := range ids {
					if name, ok := labelMap[id]; ok && name != "" {
						link[username] = append(link[username], name)
					}
				}
			}
			if err := rows.Err(); err != nil {
				return fmt.Errorf("iterate label id list: %w", err)
			}
		}
	}

	if len(link) == 0 {
		tbl, userCol, labelCol, err := detectLinkTable(ctx, db)
		if err != nil {
			return err
		}
		if tbl != "" {
			query := fmt.Sprintf(`SELECT %s, %s FROM %s`, userCol, labelCol, tbl)
			rows, err := db.QueryContext(ctx, query)
			if err != nil {
				if !isNoSuchTableErr(err) {
					return fmt.Errorf("query label link table: %w", err)
				}
			} else {
				defer rows.Close()
				for rows.Next() {
					var username string
					var labelID int
					if err := rows.Scan(&username, &labelID); err != nil {
						return fmt.Errorf("scan label link table: %w", err)
					}
					if name, ok := labelMap[labelID]; ok && name != "" {
						link[username] = append(link[username], name)
					}
				}
				if err := rows.Err(); err != nil {
					return fmt.Errorf("iterate label link table: %w", err)
				}
			}
		}
	}

	if len(link) == 0 {
		hasDescription, err := columnExists(ctx, db, "contact", "description")
		if err != nil && !isNoSuchTableErr(err) {
			return fmt.Errorf("check description column: %w", err)
		}
		var descStmt *sql.Stmt
		if hasDescription {
			if descStmt, err = db.PrepareContext(ctx, `SELECT description FROM contact WHERE username = ?`); err != nil && !isNoSuchTableErr(err) {
				return fmt.Errorf("prepare description query: %w", err)
			}
		}
		if descStmt != nil {
			defer descStmt.Close()
		}
		for _, c := range contacts {
			if looksLikeClient(c.Remark) {
				c.Labels = append(c.Labels, "客户")
				continue
			}
			if descStmt != nil {
				var description sql.NullString
				if err := descStmt.QueryRowContext(ctx, c.UserName).Scan(&description); err != nil {
					if !errors.Is(err, sql.ErrNoRows) {
						return fmt.Errorf("query description: %w", err)
					}
				} else if description.Valid && looksLikeClient(description.String) {
					c.Labels = append(c.Labels, "客户")
				}
			}
		}
		return nil
	}

	for _, c := range contacts {
		if names, ok := link[c.UserName]; ok && len(names) > 0 {
			c.Labels = append(c.Labels, dedup(names)...)
		}
	}

	return nil
}

func loadAllLabels(ctx context.Context, db *sql.DB) (map[int]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT label_id_, label_name_ FROM contact_label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labels := map[int]string{}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan contact_label: %w", err)
		}
		labels[id] = name
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate contact_label: %w", err)
	}
	return labels, nil
}

func columnExists(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	query := fmt.Sprintf(`PRAGMA table_info(%s)`, table)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	lower := strings.ToLower(column)
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			dfltValue  sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &dfltValue, &pk); err != nil {
			return false, fmt.Errorf("scan pragma table_info: %w", err)
		}
		if strings.EqualFold(name, lower) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate pragma table_info: %w", err)
	}
	return false, nil
}

func detectLinkTable(ctx context.Context, db *sql.DB) (string, string, string, error) {
	candidates := []struct {
		table    string
		userCol  string
		labelCol string
	}{
		{"rcontact_label", "username", "label_id_"},
		{"contact_label_map", "username", "label_id_"},
		{"contact2label", "username", "label_id_"},
	}

	for _, candidate := range candidates {
		exists, err := tableExists(ctx, db, candidate.table)
		if err != nil {
			if isNoSuchTableErr(err) {
				continue
			}
			return "", "", "", fmt.Errorf("check table %s: %w", candidate.table, err)
		}
		if !exists {
			continue
		}
		ok, err := columnsExist(ctx, db, candidate.table, candidate.userCol, candidate.labelCol)
		if err != nil {
			return "", "", "", err
		}
		if ok {
			return candidate.table, candidate.userCol, candidate.labelCol, nil
		}
	}

	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name LIKE '%label%'`)
	if err != nil {
		if isNoSuchTableErr(err) {
			return "", "", "", nil
		}
		return "", "", "", fmt.Errorf("query sqlite_master: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return "", "", "", fmt.Errorf("scan sqlite_master: %w", err)
		}
		ok, err := columnsExist(ctx, db, table, "username", "label_id_")
		if err != nil {
			if isNoSuchTableErr(err) {
				continue
			}
			return "", "", "", err
		}
		if ok {
			return table, "username", "label_id_", nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", "", "", fmt.Errorf("iterate sqlite_master: %w", err)
	}

	return "", "", "", nil
}

func tableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	query := `SELECT 1 FROM sqlite_master WHERE type='table' AND name=?`
	row := db.QueryRowContext(ctx, query, table)
	var one int
	if err := row.Scan(&one); err != nil {
		if isNoSuchTableErr(err) {
			return false, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func columnsExist(ctx context.Context, db *sql.DB, table string, columns ...string) (bool, error) {
	exists, err := tableExists(ctx, db, table)
	if err != nil || !exists {
		return false, err
	}
	query := fmt.Sprintf(`PRAGMA table_info(%s)`, table)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	found := map[string]bool{}
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			dfltValue  sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &dfltValue, &pk); err != nil {
			return false, fmt.Errorf("scan pragma table_info: %w", err)
		}
		found[strings.ToLower(name)] = true
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate pragma table_info: %w", err)
	}
	for _, column := range columns {
		if !found[strings.ToLower(column)] {
			return false, nil
		}
	}
	return true, nil
}

func splitIDs(raw string) []int {
	parts := strings.Split(raw, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var id int
		if _, err := fmt.Sscanf(part, "%d", &id); err == nil && id > 0 {
			result = append(result, id)
		}
	}
	return result
}

func looksLikeClient(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	normalized := strings.ToLower(value)
	switch {
	case strings.Contains(value, "客户"):
		return true
	case strings.Contains(normalized, "[客户]"):
		return true
	case strings.Contains(normalized, "#客户"):
		return true
	case strings.Contains(normalized, "customer"):
		return true
	}
	return false
}

func dedup(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func isNoSuchTableErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no such table")
}
