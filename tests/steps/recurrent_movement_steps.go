package steps

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"personal-finance/internal/model"
	"personal-finance/tests/helpers"
	"personal-finance/tests/suite"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
)

type RecurrentMovementSteps struct {
	movement   model.Movement
	movements  []model.MovementOutput
	response   *http.Response
	dateHelper *helpers.DateHelper
	apiService *helpers.ApiService
	suite      *suite.TestSuite
	movementID *uuid.UUID
	wallet     model.WalletOutput
}

func NewRecurrentMovementSteps(testSuite *suite.TestSuite) *RecurrentMovementSteps {
	return &RecurrentMovementSteps{
		suite:      testSuite,
		apiService: helpers.NewApiService(testSuite.GetServer()),
		dateHelper: helpers.NewDateHelper(),
	}
}

func (s *RecurrentMovementSteps) RegisterSteps(ctx *godog.ScenarioContext) {
	// Existing steps (mantém compatibilidade)
	ctx.Step(`^I have a recurrent movement with amount ([-0-9.]+) and description "([^"]*)"$`, s.iHaveARecurrentMovementWithAmountAndDescription)

	// New enhanced steps (com is_paid)
	ctx.Step(`^I have a recurrent movement with amount ([-0-9.]+) and description "([^"]*)" and is_paid (true|false)$`, s.iHaveARecurrentMovementWithAmountDescriptionAndPaidStatus)

	// Movement operations
	ctx.Step(`^I create the movement$`, s.iCreateTheMovement)
	ctx.Step(`^I pay the movement$`, s.iPayTheMovement)
	ctx.Step(`^the operation should be successful$`, s.theOperationShouldBeSuccessful)

	// Search operations
	ctx.Step(`^I search for movements in month ([0-9]+)$`, s.iSearchForMovementsInMonth)
	ctx.Step(`^I should find the movement with amount ([-0-9.]+)$`, s.iShouldFindTheMovementWithAmount)
	ctx.Step(`^I should not find any movement$`, s.iShouldNotFindAnyMovement)

	// Update operations
	ctx.Step(`^I update the movement in month ([0-9]+) with amount ([-0-9.]+)$`, s.iUpdateTheMovementInMonthWithAmount)
	ctx.Step(`^I update all next occurrences from month ([0-9]+) with amount ([-0-9.]+)$`, s.iUpdateAllNextOccurrencesFromMonthWithAmount)

	// Delete operations
	ctx.Step(`^I delete the movement in month ([0-9]+)$`, s.iDeleteTheMovementInMonth)
	ctx.Step(`^I delete all future occurrences from month ([0-9]+)$`, s.iDeleteAllFutureOccurrencesFromMonth)

	// Wallet validation steps
	ctx.Step(`^I get the wallet information$`, s.iGetTheWalletInformation)
	ctx.Step(`^the wallet balance should be ([-0-9.]+)$`, s.theWalletBalanceShouldBe)
}

func (s *RecurrentMovementSteps) iHaveARecurrentMovementWithAmountAndDescription(amount, description string) error {
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %v", err)
	}

	baseData := s.suite.GetBaseData()
	now := time.Now()

	s.movement = model.Movement{
		Amount:        amountFloat,
		Description:   description,
		IsRecurrent:   true,
		Date:          &now,
		UserID:        baseData.TestUserID,
		WalletID:      baseData.DefaultWallet.ID,
		CategoryID:    baseData.DefaultCategory.ID,
		SubCategoryID: baseData.DefaultSubCategory.ID,
	}
	return nil
}

// New enhanced step with is_paid support
func (s *RecurrentMovementSteps) iHaveARecurrentMovementWithAmountDescriptionAndPaidStatus(amount, description, paidStatus string) error {
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %v", err)
	}

	isPaid := paidStatus == "true"

	baseData := s.suite.GetBaseData()
	now := time.Now()

	s.movement = model.Movement{
		Amount:        amountFloat,
		Description:   description,
		IsRecurrent:   true,
		IsPaid:        isPaid,
		Date:          &now,
		UserID:        baseData.TestUserID,
		WalletID:      baseData.DefaultWallet.ID,
		CategoryID:    baseData.DefaultCategory.ID,
		SubCategoryID: baseData.DefaultSubCategory.ID,
	}
	return nil
}

func (s *RecurrentMovementSteps) iCreateTheMovement() error {
	resp, err := s.apiService.CreateMovement(s.movement)
	if err != nil {
		return fmt.Errorf("error creating movement: %v", err)
	}
	s.response = resp

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		createdMovement, err := s.apiService.ParseMovementResponse(resp)
		if err != nil {
			return fmt.Errorf("error parsing created movement: %v", err)
		}
		s.movementID = createdMovement.ID
	}

	return nil
}

func (s *RecurrentMovementSteps) iPayTheMovement() error {
	if s.movementID == nil {
		return fmt.Errorf("no movement ID available for payment")
	}

	resp, err := s.apiService.PayMovement(*s.movementID)
	if err != nil {
		return fmt.Errorf("error paying movement: %v", err)
	}
	s.response = resp

	return nil
}

func (s *RecurrentMovementSteps) theOperationShouldBeSuccessful() error {
	if s.response.StatusCode < 200 || s.response.StatusCode >= 300 {
		return fmt.Errorf("expected successful status code, got %d", s.response.StatusCode)
	}
	return nil
}

func (s *RecurrentMovementSteps) iSearchForMovementsInMonth(monthOffset string) error {
	offset, err := strconv.Atoi(monthOffset)
	if err != nil {
		return fmt.Errorf("invalid month offset: %v", err)
	}

	targetDate := s.dateHelper.AddMonths(offset)
	fromDate := time.Date(targetDate.Year(), targetDate.Month(), 1, 0, 0, 0, 0, targetDate.Location())
	toDate := fromDate.AddDate(0, 1, -1)

	resp, err := s.apiService.GetMovementsByPeriod(fromDate, toDate)
	if err != nil {
		return fmt.Errorf("error searching movements: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error response: %d", resp.StatusCode)
	}

	movements, err := s.apiService.ParseMovementsResponse(resp)
	if err != nil {
		return fmt.Errorf("error parsing movements response: %v", err)
	}

	s.movements = movements
	return nil
}

func (s *RecurrentMovementSteps) iShouldFindTheMovementWithAmount(expectedAmount string) error {
	expectedAmountFloat, err := strconv.ParseFloat(expectedAmount, 64)
	if err != nil {
		return fmt.Errorf("invalid expected amount: %v", err)
	}

	for _, movement := range s.movements {
		if movement.Amount == expectedAmountFloat {
			return nil
		}
	}

	return fmt.Errorf("movement with amount %.2f not found", expectedAmountFloat)
}

func (s *RecurrentMovementSteps) iShouldNotFindAnyMovement() error {
	if len(s.movements) > 0 {
		return fmt.Errorf("expected no movements, but found %d", len(s.movements))
	}
	return nil
}

func (s *RecurrentMovementSteps) iUpdateTheMovementInMonthWithAmount(monthOffset, newAmount string) error {
	if s.movementID == nil {
		return fmt.Errorf("no movement ID available for update")
	}

	offset, err := strconv.Atoi(monthOffset)
	if err != nil {
		return fmt.Errorf("invalid month offset: %v", err)
	}

	newAmountFloat, err := strconv.ParseFloat(newAmount, 64)
	if err != nil {
		return fmt.Errorf("invalid new amount: %v", err)
	}

	targetDate := s.dateHelper.AddMonths(offset)
	updateData := model.Movement{
		Amount: newAmountFloat,
		Date:   &targetDate,
	}

	resp, err := s.apiService.UpdateMovement(*s.movementID, updateData, targetDate)
	if err != nil {
		return fmt.Errorf("error updating movement: %v", err)
	}
	s.response = resp
	return nil
}

func (s *RecurrentMovementSteps) iUpdateAllNextOccurrencesFromMonthWithAmount(monthOffset, newAmount string) error {
	if s.movementID == nil {
		return fmt.Errorf("no movement ID available for update")
	}

	offset, err := strconv.Atoi(monthOffset)
	if err != nil {
		return fmt.Errorf("invalid month offset: %v", err)
	}

	newAmountFloat, err := strconv.ParseFloat(newAmount, 64)
	if err != nil {
		return fmt.Errorf("invalid new amount: %v", err)
	}

	targetDate := s.dateHelper.AddMonths(offset)
	updateData := model.Movement{
		Amount: newAmountFloat,
		Date:   &targetDate,
	}

	resp, err := s.apiService.UpdateAllNextMovements(*s.movementID, updateData, targetDate)
	if err != nil {
		return fmt.Errorf("error updating all next movements: %v", err)
	}
	s.response = resp
	return nil
}

func (s *RecurrentMovementSteps) iDeleteTheMovementInMonth(monthOffset string) error {
	if s.movementID == nil {
		return fmt.Errorf("no movement ID available for delete")
	}

	offset, err := strconv.Atoi(monthOffset)
	if err != nil {
		return fmt.Errorf("invalid month offset: %v", err)
	}

	targetDate := s.dateHelper.AddMonths(offset)

	resp, err := s.apiService.DeleteMovement(*s.movementID, targetDate)
	if err != nil {
		return fmt.Errorf("error deleting movement: %v", err)
	}
	s.response = resp
	return nil
}

func (s *RecurrentMovementSteps) iDeleteAllFutureOccurrencesFromMonth(monthOffset string) error {
	if s.movementID == nil {
		return fmt.Errorf("no movement ID available for delete")
	}

	offset, err := strconv.Atoi(monthOffset)
	if err != nil {
		return fmt.Errorf("invalid month offset: %v", err)
	}

	targetDate := s.dateHelper.AddMonths(offset)

	resp, err := s.apiService.DeleteAllNextMovements(*s.movementID, targetDate)
	if err != nil {
		return fmt.Errorf("error deleting future movements: %v", err)
	}
	s.response = resp
	return nil
}

// Wallet validation methods
func (s *RecurrentMovementSteps) iGetTheWalletInformation() error {
	baseData := s.suite.GetBaseData()
	walletID := baseData.DefaultWallet.ID

	resp, err := s.apiService.GetWallet(*walletID)
	if err != nil {
		return fmt.Errorf("error getting wallet information: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error response getting wallet: %d", resp.StatusCode)
	}

	wallet, err := s.apiService.ParseWalletResponse(resp)
	if err != nil {
		return fmt.Errorf("error parsing wallet response: %v", err)
	}

	s.wallet = *wallet
	return nil
}

func (s *RecurrentMovementSteps) theWalletBalanceShouldBe(expectedBalance string) error {
	expectedBalanceFloat, err := strconv.ParseFloat(expectedBalance, 64)
	if err != nil {
		return fmt.Errorf("invalid expected balance: %v", err)
	}

	if s.wallet.Balance != expectedBalanceFloat {
		return fmt.Errorf("expected wallet balance %.2f, but got %.2f", expectedBalanceFloat, s.wallet.Balance)
	}

	return nil
}
