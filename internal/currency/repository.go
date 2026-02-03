package currency

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for currency
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new currency repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetActiveCurrencies retrieves all active currencies
func (r *Repository) GetActiveCurrencies(ctx context.Context) ([]*Currency, error) {
	query := `
		SELECT code, name, symbol, decimal_places, is_active, created_at
		FROM currencies
		WHERE is_active = true
		ORDER BY code
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get currencies: %w", err)
	}
	defer rows.Close()

	currencies := make([]*Currency, 0)
	for rows.Next() {
		c := &Currency{}
		err := rows.Scan(&c.Code, &c.Name, &c.Symbol, &c.DecimalPlaces, &c.IsActive, &c.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan currency: %w", err)
		}
		currencies = append(currencies, c)
	}

	return currencies, nil
}

// GetCurrencyByCode retrieves a currency by its code
func (r *Repository) GetCurrencyByCode(ctx context.Context, code string) (*Currency, error) {
	query := `
		SELECT code, name, symbol, decimal_places, is_active, created_at
		FROM currencies
		WHERE code = $1
	`

	c := &Currency{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&c.Code, &c.Name, &c.Symbol, &c.DecimalPlaces, &c.IsActive, &c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency: %w", err)
	}

	return c, nil
}

// GetLatestExchangeRate retrieves the latest valid exchange rate
func (r *Repository) GetLatestExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*ExchangeRate, error) {
	query := `
		SELECT id, from_currency, to_currency, rate, inverse_rate, source,
		       fetched_at, valid_until, created_at
		FROM exchange_rates
		WHERE from_currency = $1 AND to_currency = $2
		  AND valid_until > NOW()
		ORDER BY fetched_at DESC
		LIMIT 1
	`

	rate := &ExchangeRate{}
	err := r.db.QueryRow(ctx, query, fromCurrency, toCurrency).Scan(
		&rate.ID, &rate.FromCurrency, &rate.ToCurrency, &rate.Rate,
		&rate.InverseRate, &rate.Source, &rate.FetchedAt, &rate.ValidUntil, &rate.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	return rate, nil
}

// GetExchangeRateByID retrieves an exchange rate by ID
func (r *Repository) GetExchangeRateByID(ctx context.Context, id uuid.UUID) (*ExchangeRate, error) {
	query := `
		SELECT id, from_currency, to_currency, rate, inverse_rate, source,
		       fetched_at, valid_until, created_at
		FROM exchange_rates
		WHERE id = $1
	`

	rate := &ExchangeRate{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rate.ID, &rate.FromCurrency, &rate.ToCurrency, &rate.Rate,
		&rate.InverseRate, &rate.Source, &rate.FetchedAt, &rate.ValidUntil, &rate.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	return rate, nil
}

// CreateExchangeRate creates a new exchange rate
func (r *Repository) CreateExchangeRate(ctx context.Context, rate *ExchangeRate) error {
	query := `
		INSERT INTO exchange_rates (id, from_currency, to_currency, rate, inverse_rate,
		                            source, fetched_at, valid_until)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`

	rate.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		rate.ID, rate.FromCurrency, rate.ToCurrency, rate.Rate,
		rate.InverseRate, rate.Source, rate.FetchedAt, rate.ValidUntil,
	).Scan(&rate.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create exchange rate: %w", err)
	}

	return nil
}

// BulkCreateExchangeRates creates multiple exchange rates in a batch
func (r *Repository) BulkCreateExchangeRates(ctx context.Context, rates []*ExchangeRate) error {
	if len(rates) == 0 {
		return nil
	}

	// Use a transaction for bulk insert
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, rate := range rates {
		rate.ID = uuid.New()
		_, err := tx.Exec(ctx, `
			INSERT INTO exchange_rates (id, from_currency, to_currency, rate, inverse_rate,
			                            source, fetched_at, valid_until)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, rate.ID, rate.FromCurrency, rate.ToCurrency, rate.Rate,
			rate.InverseRate, rate.Source, rate.FetchedAt, rate.ValidUntil)

		if err != nil {
			return fmt.Errorf("failed to create exchange rate: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetAllExchangeRatesFromBase retrieves all exchange rates from a base currency
func (r *Repository) GetAllExchangeRatesFromBase(ctx context.Context, baseCurrency string) ([]*ExchangeRate, error) {
	query := `
		SELECT DISTINCT ON (to_currency)
		       id, from_currency, to_currency, rate, inverse_rate, source,
		       fetched_at, valid_until, created_at
		FROM exchange_rates
		WHERE from_currency = $1 AND valid_until > NOW()
		ORDER BY to_currency, fetched_at DESC
	`

	rows, err := r.db.Query(ctx, query, baseCurrency)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rates: %w", err)
	}
	defer rows.Close()

	rates := make([]*ExchangeRate, 0)
	for rows.Next() {
		rate := &ExchangeRate{}
		err := rows.Scan(
			&rate.ID, &rate.FromCurrency, &rate.ToCurrency, &rate.Rate,
			&rate.InverseRate, &rate.Source, &rate.FetchedAt, &rate.ValidUntil, &rate.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exchange rate: %w", err)
		}
		rates = append(rates, rate)
	}

	return rates, nil
}

// InvalidateExchangeRates marks exchange rates as expired
func (r *Repository) InvalidateExchangeRates(ctx context.Context, fromCurrency string) error {
	query := `
		UPDATE exchange_rates
		SET valid_until = NOW()
		WHERE from_currency = $1 AND valid_until > NOW()
	`

	_, err := r.db.Exec(ctx, query, fromCurrency)
	if err != nil {
		return fmt.Errorf("failed to invalidate exchange rates: %w", err)
	}

	return nil
}

// CreateCurrency creates a new currency
func (r *Repository) CreateCurrency(ctx context.Context, currency *Currency) error {
	query := `
		INSERT INTO currencies (code, name, symbol, decimal_places, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`

	err := r.db.QueryRow(ctx, query,
		currency.Code, currency.Name, currency.Symbol,
		currency.DecimalPlaces, currency.IsActive,
	).Scan(&currency.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create currency: %w", err)
	}

	return nil
}

// UpdateCurrency updates a currency
func (r *Repository) UpdateCurrency(ctx context.Context, currency *Currency) error {
	query := `
		UPDATE currencies
		SET name = $1, symbol = $2, decimal_places = $3, is_active = $4
		WHERE code = $5
	`

	_, err := r.db.Exec(ctx, query,
		currency.Name, currency.Symbol, currency.DecimalPlaces,
		currency.IsActive, currency.Code,
	)
	if err != nil {
		return fmt.Errorf("failed to update currency: %w", err)
	}

	return nil
}

// CleanupExpiredRates removes exchange rates that have been expired for more than the given duration
func (r *Repository) CleanupExpiredRates(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM exchange_rates
		WHERE valid_until < $1
	`

	cutoff := time.Now().Add(-olderThan)
	tag, err := r.db.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup exchange rates: %w", err)
	}

	return tag.RowsAffected(), nil
}
