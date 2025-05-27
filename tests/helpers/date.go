package helpers

import "time"

type DateHelper struct {
	baseDate time.Time
}

func NewDateHelper() *DateHelper {
	return &DateHelper{
		baseDate: time.Now(),
	}
}

func (h *DateHelper) CurrentMonth() time.Time {
	return h.baseDate
}

func (h *DateHelper) NextMonth() time.Time {
	return h.baseDate.AddDate(0, 1, 0)
}

func (h *DateHelper) PreviousMonth() time.Time {
	return h.baseDate.AddDate(0, -1, 0)
}

func (h *DateHelper) AddMonths(months int) time.Time {
	return h.baseDate.AddDate(0, months, 0)
}
