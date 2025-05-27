package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"personal-finance/internal/model"

	"github.com/google/uuid"
)

// ApiService centraliza todas as chamadas HTTP para a API
type ApiService struct {
	server *httptest.Server
	client *http.Client
}

// NewApiService cria nova instância do service
func NewApiService(server *httptest.Server) *ApiService {
	return &ApiService{
		server: server,
		client: &http.Client{},
	}
}

func (a *ApiService) CreateMovement(movement model.Movement) (*http.Response, error) {
	jsonData, err := json.Marshal(movement)
	if err != nil {
		return nil, fmt.Errorf("error marshaling movement: %v", err)
	}

	url := fmt.Sprintf("%s/movements/simple", a.server.URL)
	return http.Post(url, "application/json", bytes.NewBuffer(jsonData))
}

func (a *ApiService) PayMovement(movementID uuid.UUID) (*http.Response, error) {
	url := fmt.Sprintf("%s/movements/%s/pay", a.server.URL, movementID.String())
	return http.Post(url, "application/json", nil)
}

func (a *ApiService) GetMovementsByPeriod(from, to time.Time) (*http.Response, error) {
	url := fmt.Sprintf("%s/movements/period?from=%s&to=%s",
		a.server.URL,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"))

	return http.Get(url)
}

func (a *ApiService) UpdateMovement(movementID uuid.UUID, movement model.Movement, date time.Time) (*http.Response, error) {
	jsonData, err := json.Marshal(movement)
	if err != nil {
		return nil, fmt.Errorf("error marshaling update data: %v", err)
	}

	url := fmt.Sprintf("%s/movements/%s?date=%s",
		a.server.URL,
		movementID.String(),
		date.Format("2006-01-02"))

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return a.client.Do(req)
}

func (a *ApiService) UpdateAllNextMovements(movementID uuid.UUID, movement model.Movement, date time.Time) (*http.Response, error) {
	jsonData, err := json.Marshal(movement)
	if err != nil {
		return nil, fmt.Errorf("error marshaling update data: %v", err)
	}

	url := fmt.Sprintf("%s/movements/%s/all-next?date=%s",
		a.server.URL,
		movementID.String(),
		date.Format("2006-01-02"))

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return a.client.Do(req)
}

func (a *ApiService) DeleteMovement(movementID uuid.UUID, date time.Time) (*http.Response, error) {
	url := fmt.Sprintf("%s/movements/%s?date=%s",
		a.server.URL,
		movementID.String(),
		date.Format("2006-01-02"))

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating delete request: %v", err)
	}

	return a.client.Do(req)
}

func (a *ApiService) DeleteAllNextMovements(movementID uuid.UUID, date time.Time) (*http.Response, error) {
	url := fmt.Sprintf("%s/movements/%s/all-next?date=%s",
		a.server.URL,
		movementID.String(),
		date.Format("2006-01-02"))

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating delete request: %v", err)
	}

	return a.client.Do(req)
}

func (a *ApiService) GetWallet(walletID uuid.UUID) (*http.Response, error) {
	url := fmt.Sprintf("%s/wallets/%s", a.server.URL, walletID.String())
	return http.Get(url)
}

func (a *ApiService) ParseMovementResponse(response *http.Response) (*model.MovementOutput, error) {
	var movement model.MovementOutput
	if err := json.NewDecoder(response.Body).Decode(&movement); err != nil {
		return nil, fmt.Errorf("error decoding movement response: %v", err)
	}
	return &movement, nil
}

func (a *ApiService) ParseMovementsResponse(response *http.Response) ([]model.MovementOutput, error) {
	var movements []model.MovementOutput
	if err := json.NewDecoder(response.Body).Decode(&movements); err != nil {
		return nil, fmt.Errorf("error decoding movements response: %v", err)
	}
	return movements, nil
}

func (a *ApiService) ParseWalletResponse(response *http.Response) (*model.WalletOutput, error) {
	var wallet model.WalletOutput
	if err := json.NewDecoder(response.Body).Decode(&wallet); err != nil {
		return nil, fmt.Errorf("error decoding wallet response: %v", err)
	}
	return &wallet, nil
}
