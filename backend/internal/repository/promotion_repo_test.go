package repository

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestPromotionDetailsBindParameters(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	filter := service.PromotionFilter{
		Start: start, End: start.AddDate(0, 0, 1), Timezone: "Asia/Shanghai",
		ChannelID: 3, Search: "query", Status: "paid", Page: 2, PageSize: 20,
	}

	tests := []struct {
		kind          string
		queryPattern  string
		wantArguments []driver.Value
	}{
		{
			kind:         "expenses",
			queryPattern: `WITH[\s\S]*promotion_expenses[\s\S]*LIMIT \$5 OFFSET \$6[\s\S]*'page',\$8::int,'page_size',\$5::int,'timezone',\$7::text`,
			wantArguments: []driver.Value{
				filter.Start, filter.End, filter.ChannelID, filter.Search,
				filter.PageSize, 20, filter.Timezone, filter.Page,
			},
		},
		{
			kind:         "payments",
			queryPattern: `WITH[\s\S]*payment_orders[\s\S]*LIMIT \$5 OFFSET \$6[\s\S]*'page',\$8::int,'page_size',\$5::int,'timezone',\$7::text`,
			wantArguments: []driver.Value{
				filter.Start, filter.End, filter.ChannelID, filter.Search,
				filter.PageSize, 20, filter.Timezone, filter.Page,
			},
		},
		{
			kind:         "users",
			queryPattern: `WITH[\s\S]*promotion_attributions[\s\S]*status=\$5[\s\S]*LIMIT \$6 OFFSET \$7[\s\S]*'page',\$9::int,'page_size',\$6::int,'timezone',\$8::text`,
			wantArguments: []driver.Value{
				filter.Start, filter.End, filter.ChannelID, filter.Search, filter.Status,
				filter.PageSize, 20, filter.Timezone, filter.Page,
			},
		},
		{
			kind:         "visitors",
			queryPattern: `WITH[\s\S]*visitor_cohorts[\s\S]*status=\$5[\s\S]*LIMIT \$6 OFFSET \$7[\s\S]*'page',\$9::int,'page_size',\$6::int,'timezone',\$8::text`,
			wantArguments: []driver.Value{
				filter.Start, filter.End, filter.ChannelID, filter.Search, filter.Status,
				filter.PageSize, 20, filter.Timezone, filter.Page,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			mock.ExpectQuery(tt.queryPattern).
				WithArgs(tt.wantArguments...).
				WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow([]byte(`{"items":[]}`)))

			_, err = (&promotionRepository{db: db}).Details(context.Background(), tt.kind, filter)
			if err != nil {
				t.Fatalf("Details() error = %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
