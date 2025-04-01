package model

import "time"

type Rate struct {
	ID        int64     `db:"id"`
	Symbol    string    `db:"symbol"`
	Ask       float64   `db:"ask"`
	Bid       float64   `db:"bid"`
	Timestamp time.Time `db:"timestamp"`
	CreatedAt time.Time `db:"created_at"`
}
