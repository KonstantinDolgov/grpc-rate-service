package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"studentgit.kata.academy/KonstantinDolgov/grpc-rate-service/internal/model"
)

func TestNewRepository(t *testing.T) {
	// Arrange
	logger := zap.NewNop()

	// Act & Assert
	t.Run("should return error for invalid DSN", func(t *testing.T) {
		repo, err := NewRepository("invalid-dsn", logger)
		assert.Error(t, err)
		assert.Nil(t, repo)
	})
}

func TestSaveRate(t *testing.T) {
	// Arrange
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	logger := zap.NewNop()
	repo := &Repository{
		db:     db,
		logger: logger,
	}

	ctx := context.Background()
	rate := model.Rate{
		Symbol:    "BTC-USDT",
		Ask:       40000.5,
		Bid:       39999.5,
		Timestamp: time.Now().UTC(),
		CreatedAt: time.Now(),
	}

	t.Run("successful save", func(t *testing.T) {
		// Ожидаем выполнение SQL запроса с правильными аргументами
		mock.ExpectExec("INSERT INTO rates").
			WithArgs(rate.Symbol, rate.Ask, rate.Bid, rate.Timestamp, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Act
		err := repo.SaveRate(ctx, rate)

		// Assert
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		// Ожидаем выполнение SQL запроса с ошибкой
		dbError := errors.New("database error")
		mock.ExpectExec("INSERT INTO rates").
			WithArgs(rate.Symbol, rate.Ask, rate.Bid, rate.Timestamp, sqlmock.AnyArg()).
			WillReturnError(dbError)

		// Act
		err := repo.SaveRate(ctx, rate)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute insert query")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetLatestRate(t *testing.T) {
	// Arrange
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	logger := zap.NewNop()
	repo := &Repository{
		db:     db,
		logger: logger,
	}

	ctx := context.Background()
	symbol := "BTC-USDT"
	now := time.Now().UTC()

	t.Run("successful retrieval", func(t *testing.T) {
		// Создаем заглушку для результата запроса
		rows := sqlmock.NewRows([]string{"id", "symbol", "ask", "bid", "timestamp", "created_at"}).
			AddRow(1, symbol, 40000.5, 39999.5, now, now)

		// Ожидаем выполнение SQL запроса
		mock.ExpectQuery("SELECT (.+) FROM rates").
			WithArgs(symbol).
			WillReturnRows(rows)

		// Act
		rate, err := repo.GetLatestRate(ctx, symbol)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, rate)
		assert.Equal(t, int64(1), rate.ID)
		assert.Equal(t, symbol, rate.Symbol)
		assert.Equal(t, 40000.5, rate.Ask)
		assert.Equal(t, 39999.5, rate.Bid)
		assert.Equal(t, now, rate.Timestamp)
		assert.Equal(t, now, rate.CreatedAt)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows found", func(t *testing.T) {
		// Ожидаем выполнение SQL запроса без результатов
		mock.ExpectQuery("SELECT (.+) FROM rates").
			WithArgs(symbol).
			WillReturnError(sql.ErrNoRows)

		// Act
		rate, err := repo.GetLatestRate(ctx, symbol)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, rate)
		assert.Equal(t, sql.ErrNoRows, errors.Unwrap(err))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		// Ожидаем выполнение SQL запроса с ошибкой
		dbError := errors.New("database error")
		mock.ExpectQuery("SELECT (.+) FROM rates").
			WithArgs(symbol).
			WillReturnError(dbError)

		// Act
		rate, err := repo.GetLatestRate(ctx, symbol)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, rate)
		assert.Contains(t, err.Error(), "failed to get latest rate")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClose(t *testing.T) {
	// Arrange
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	logger := zap.NewNop()
	repo := &Repository{
		db:     db,
		logger: logger,
	}

	t.Run("successful close", func(t *testing.T) {
		// Ожидаем закрытие соединения
		mock.ExpectClose()

		// Act
		err := repo.Close()

		// Assert
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("close error", func(t *testing.T) {
		// Создаем новое соединение и мок, так как предыдущее уже закрыто
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}

		repo := &Repository{
			db:     db,
			logger: logger,
		}

		// Ожидаем ошибку при закрытии
		dbError := errors.New("close error")
		mock.ExpectClose().WillReturnError(dbError)

		// Act
		err = repo.Close()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to close database connection")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
