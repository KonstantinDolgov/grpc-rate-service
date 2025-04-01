-- +goose Up
-- +goose StatementBegin
CREATE TABLE rates (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    ask DECIMAL(20, 10) NOT NULL,
    bid DECIMAL(20, 10) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rates_symbol ON rates(symbol);
CREATE INDEX idx_rates_timestamp ON rates(timestamp);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS rates;
-- +goose StatementEnd
