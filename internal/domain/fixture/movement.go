package fixture

import (
	"github.com/google/uuid"
	"personal-finance/internal/model"
	"time"
)

var (
	idString = "411e010e-187a-475f-a675-4a1575a95ed8"
)

type MovementOptions func(movement *model.Movement)

func WithAmount(amount float64) MovementOptions {
	return func(mov *model.Movement) {
		mov.Amount = amount
	}
}

func WithDescription(description string) MovementOptions {
	return func(mov *model.Movement) {
		mov.Description = description
	}
}

func WithDate(date time.Time) MovementOptions {
	return func(mov *model.Movement) {
		mov.Date = &date
	}
}

func MockMovement(options ...MovementOptions) model.Movement {
	id, _ := uuid.Parse(idString)
	date := time.Date(2021, 10, 10, 0, 0, 0, 0, time.UTC)

	movement := model.Movement{
		ID:          &id,
		Description: "Car",
		Amount:      100,
		Date:        &date,
	}

	for _, opt := range options {
		opt(&movement)
	}
	return movement
}
