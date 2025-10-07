package v4

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/sjzar/chatlog/internal/model"
)

func TestFillContactLabelsV4(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, db *sql.DB) []*model.ContactV4
		want  map[string][]string
	}{
		{
			name: "LabelIDList column",
			setup: func(t *testing.T, db *sql.DB) []*model.ContactV4 {
				execAll(t, db,
					`CREATE TABLE contact (username TEXT, LabelIDList TEXT, remark TEXT, description TEXT);`,
					`CREATE TABLE contact_label (label_id_ INTEGER, label_name_ TEXT);`,
					`INSERT INTO contact_label(label_id_, label_name_) VALUES (1, '客户'), (2, 'VIP');`,
					`INSERT INTO contact(username, LabelIDList, remark) VALUES ('alice', '1,2', 'Alice'), ('bob', NULL, 'Bob');`,
				)
				return []*model.ContactV4{
					{UserName: "alice", Remark: "Alice"},
					{UserName: "bob", Remark: "Bob"},
				}
			},
			want: map[string][]string{
				"alice": []string{"客户", "VIP"},
				"bob":   nil,
			},
		},
		{
			name: "Mapping table fallback",
			setup: func(t *testing.T, db *sql.DB) []*model.ContactV4 {
				execAll(t, db,
					`CREATE TABLE contact (username TEXT, remark TEXT);`,
					`CREATE TABLE contact_label (label_id_ INTEGER, label_name_ TEXT);`,
					`CREATE TABLE rcontact_label (username TEXT, label_id_ INTEGER);`,
					`INSERT INTO contact_label(label_id_, label_name_) VALUES (1, '朋友'), (2, '供应商');`,
					`INSERT INTO rcontact_label(username, label_id_) VALUES ('carol', 1), ('dave', 2), ('dave', 1);`,
				)
				return []*model.ContactV4{
					{UserName: "carol", Remark: ""},
					{UserName: "dave", Remark: ""},
				}
			},
			want: map[string][]string{
				"carol": []string{"朋友"},
				"dave":  []string{"供应商", "朋友"},
			},
		},
		{
			name: "Heuristic fallback",
			setup: func(t *testing.T, db *sql.DB) []*model.ContactV4 {
				execAll(t, db,
					`CREATE TABLE contact (username TEXT, remark TEXT, description TEXT);`,
					`INSERT INTO contact(username, remark, description) VALUES ('eva', '张三-客户', NULL), ('frank', '', '潜在客户'), ('gina', '', '');`,
				)
				return []*model.ContactV4{
					{UserName: "eva", Remark: "张三-客户"},
					{UserName: "frank", Remark: ""},
					{UserName: "gina", Remark: ""},
				}
			},
			want: map[string][]string{
				"eva":   []string{"客户"},
				"frank": []string{"客户"},
				"gina":  nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := sql.Open("sqlite3", ":memory:")
			if err != nil {
				t.Fatalf("open sqlite: %v", err)
			}
			t.Cleanup(func() {
				_ = db.Close()
			})

			contacts := tt.setup(t, db)
			ctx := context.Background()
			if err := fillContactLabelsV4(ctx, db, contacts); err != nil {
				t.Fatalf("fillContactLabelsV4 returned error: %v", err)
			}

			for _, contact := range contacts {
				want := tt.want[contact.UserName]
				if !reflect.DeepEqual(contact.Labels, want) {
					t.Errorf("labels mismatch for %s: got %v, want %v", contact.UserName, contact.Labels, want)
				}
			}
		})
	}
}

func execAll(t *testing.T, db *sql.DB, statements ...string) {
	t.Helper()
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
}
