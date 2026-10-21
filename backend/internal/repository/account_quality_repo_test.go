package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"testing"
)

func TestAccountQualityVersionSurvivesSchedulerProjection(t *testing.T) {
	full, meta, err := marshalSchedulerCacheAccount(service.Account{ID: 7, QualityConfigVersion: 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{full, meta} {
		var account service.Account
		if err := json.Unmarshal(data, &account); err != nil {
			t.Fatal(err)
		}
		if account.QualityConfigVersion != 3 {
			t.Fatal("scheduler projection discarded configuration version")
		}
	}
}

func TestAccountQualityCancelledLeaseCannotPersistOrResurrect(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT NOT cancel_requested.*FOR UPDATE").WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows([]string{"active"}).AddRow(false))
	mock.ExpectRollback()
	repo := &accountQualityRepo{db: db}
	err = repo.AddResult(context.Background(), &service.AccountQualityResult{RunID: 5})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expired or canceled lease must reject result: %v", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
