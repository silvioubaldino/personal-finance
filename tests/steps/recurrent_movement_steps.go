package steps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	server     *httptest.Server
	suite      *suite.TestSuite
	lastError  error
	movementID *uuid.UUID
}

func NewRecurrentMovementSteps(testSuite *suite.TestSuite) *RecurrentMovementSteps {
	return &RecurrentMovementSteps{
		suite:      testSuite,
		server:     testSuite.GetServer(),
		dateHelper: helpers.NewDateHelper(),
	}
}

func (s *RecurrentMovementSteps) RegisterSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^I have a recurrent movement with amount ([0-9.]+) and description "([^"]*)"$`, s.iHaveARecurrentMovementWithAmountAndDescription)
	ctx.Step(`^I create the movement$`, s.iCreateTheMovement)
	ctx.Step(`^the operation should be successful$`, s.theOperationShouldBeSuccessful)
	ctx.Step(`^I search for movements in month ([0-9]+)$`, s.iSearchForMovementsInMonth)
	ctx.Step(`^I should find the movement with amount ([0-9.]+)$`, s.iShouldFindTheMovementWithAmount)
	ctx.Step(`^I should not find any movement$`, s.iShouldNotFindAnyMovement)
	ctx.Step(`^I update the movement in month ([0-9]+) with amount ([0-9.]+)$`, s.iUpdateTheMovementInMonthWithAmount)
	ctx.Step(`^I update all next occurrences from month ([0-9]+) with amount ([0-9.]+)$`, s.iUpdateAllNextOccurrencesFromMonthWithAmount)
	ctx.Step(`^I delete the movement in month ([0-9]+)$`, s.iDeleteTheMovementInMonth)
	ctx.Step(`^I delete all future occurrences from month ([0-9]+)$`, s.iDeleteAllFutureOccurrencesFromMonth)
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
		TypePaymentID: baseData.DefaultTypePayment.ID,
	}
	return nil
}

func (s *RecurrentMovementSteps) iCreateTheMovement() error {
	jsonData, err := json.Marshal(s.movement)
	if err != nil {
		return fmt.Errorf("error marshaling movement: %v", err)
	}

	resp, err := http.Post(s.server.URL+"/movements/simple", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating movement: %v", err)
	}
	s.response = resp

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var createdMovement model.MovementOutput
		if err := json.NewDecoder(resp.Body).Decode(&createdMovement); err != nil {
			return fmt.Errorf("error decoding created movement: %v", err)
		}
		s.movementID = createdMovement.ID
	}

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

	url := fmt.Sprintf("%s/movements/period?from=%s&to=%s",
		s.server.URL,
		fromDate.Format("2006-01-02"),
		toDate.Format("2006-01-02"))

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error searching movements: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error response: %d", resp.StatusCode)
	}

	var movements []model.MovementOutput
	if err := json.NewDecoder(resp.Body).Decode(&movements); err != nil {
		return fmt.Errorf("error decoding response: %v", err)
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

	updateData := model.Movement{
		Amount: newAmountFloat,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("error marshaling update data: %v", err)
	}

	targetDate := s.dateHelper.AddMonths(offset)
	url := fmt.Sprintf("%s/movements/%s?date=%s",
		s.server.URL,
		s.movementID.String(),
		targetDate.Format("2006-01-02"))

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
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

	updateData := model.Movement{
		Amount: newAmountFloat,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("error marshaling update data: %v", err)
	}

	targetDate := s.dateHelper.AddMonths(offset)
	url := fmt.Sprintf("%s/movements/%s/all-next?date=%s",
		s.server.URL,
		s.movementID.String(),
		targetDate.Format("2006-01-02"))

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
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
	url := fmt.Sprintf("%s/movements/%s?date=%s",
		s.server.URL,
		s.movementID.String(),
		targetDate.Format("2006-01-02"))

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
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
	url := fmt.Sprintf("%s/movements/%s/all-next?date=%s",
		s.server.URL,
		s.movementID.String(),
		targetDate.Format("2006-01-02"))

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error deleting future movements: %v", err)
	}
	s.response = resp
	return nil
}
