package domain

import (
	"time"

	"github.com/google/uuid"
)

type Movement struct {
	ID             *uuid.UUID          `json:"id,omitempty" gorm:"primaryKey"`
	Description    string              `json:"description,omitempty"`
	Amount         float64             `json:"amount"`
	Date           *time.Time          `json:"date"`
	UserID         string              `json:"user_id"`
	IsPaid         bool                `json:"is_paid"`
	IsRecurrent    bool                `json:"is_recurrent"`
	RecurrentID    *uuid.UUID          `json:"recurrent_id"`
	CreditCardInfo *CreditCardMovement `json:"credit_card_info,omitempty"`
	WalletID       *uuid.UUID          `json:"wallet_id,omitempty"`
	Wallet         Wallet              `json:"wallets,omitempty"`
	TypePayment    TypePayment         `json:"type_payment,omitempty"`
	CategoryID     *uuid.UUID          `json:"category_id,omitempty"`
	Category       Category            `json:"categories,omitempty"`
	SubCategoryID  *uuid.UUID          `json:"sub_category_id,omitempty"`
	SubCategory    SubCategory         `json:"sub_categories,omitempty"`
	DateCreate     time.Time           `json:"date_create"`
	DateUpdate     time.Time           `json:"date_update"`
}

type CreditCardMovement struct {
	InvoiceID          *uuid.UUID `json:"invoice_id,omitempty"`
	CreditCardID       *uuid.UUID `json:"credit_card_id,omitempty"`
	InstallmentGroupID *uuid.UUID `json:"installment_group_id,omitempty"`
	InstallmentNumber  *int       `json:"installment_number,omitempty"`
	TotalInstallments  *int       `json:"total_installments,omitempty"`
}

func (m Movement) ShouldCreateRecurrent() bool {
	return m.IsRecurrent && m.RecurrentID == nil
}

type MovementList []Movement

func (ml MovementList) GetPaidMovements() MovementList {
	var paidList MovementList
	for _, movement := range ml {
		if movement.IsPaid {
			paidList = append(paidList, movement)
		}
	}
	return paidList
}

func (ml MovementList) GetExpenseMovements() MovementList {
	var expenseList MovementList
	for _, movement := range ml {
		if movement.Amount < 0 {
			expenseList = append(expenseList, movement)
		}
	}
	return expenseList
}

func (ml MovementList) GetIncomeMovements() MovementList {
	var incomeList MovementList
	for _, movement := range ml {
		if movement.Amount > 0 {
			incomeList = append(incomeList, movement)
		}
	}
	return incomeList
}

func (ml MovementList) GetSumByCategory() map[*uuid.UUID]float64 {
	m := make(map[*uuid.UUID]float64)
	for _, movement := range ml {
		if _, ok := m[movement.CategoryID]; !ok {
			m[movement.Category.ID] = movement.Amount
		} else {
			m[movement.Category.ID] += movement.Amount
		}
	}
	return m
}

func (m Movement) ReverseAmount() float64 {
	return -m.Amount
}

func (m Movement) IsCreditCardMovement() bool {
	return m.TypePayment == TypePaymentCreditCard
}

func (m Movement) IsInstallmentMovement() bool {
	return m.CreditCardInfo != nil &&
		m.CreditCardInfo.InstallmentNumber != nil &&
		m.CreditCardInfo.TotalInstallments != nil
}

func (m Movement) BuildInstallmentMovement(installmentNumber int, date time.Time) Movement {
	id := uuid.New()
	return Movement{
		ID:          &id,
		Description: m.Description,
		Amount:      m.Amount,
		Date:        &date,
		UserID:      m.UserID,
		IsPaid:      m.IsPaid,
		IsRecurrent: m.IsRecurrent,
		CreditCardInfo: &CreditCardMovement{
			CreditCardID:       m.CreditCardInfo.CreditCardID,
			InstallmentGroupID: m.CreditCardInfo.InstallmentGroupID,
			InstallmentNumber:  &installmentNumber,
			TotalInstallments:  m.CreditCardInfo.TotalInstallments,
		},
		WalletID:      m.WalletID,
		TypePayment:   m.TypePayment,
		CategoryID:    m.CategoryID,
		SubCategoryID: m.SubCategoryID,
	}
}

func (m Movement) GenerateInstallmentMovements() MovementList {
	if !m.IsInstallmentMovement() {
		return MovementList{}
	}

	remainingInstallments := *m.CreditCardInfo.TotalInstallments - *m.CreditCardInfo.InstallmentNumber
	if remainingInstallments < 0 {
		return MovementList{}
	}

	groupID := uuid.New()

	m.CreditCardInfo.InstallmentGroupID = &groupID
	movements := MovementList{m}

	installment := *m.CreditCardInfo.InstallmentNumber
	for i := 0; i < remainingInstallments; i++ {
		installmentDate := m.Date.AddDate(0, i+1, 0)
		installment++
		movements = append(movements, m.BuildInstallmentMovement(installment, installmentDate))
	}

	return movements
}
