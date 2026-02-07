package domain

import "time"

type Service struct {
	Name      string
	Server    string
	Domain    string
	Status    string
	SpecJSON  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
