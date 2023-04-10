package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"personal-finance/internal/domain/movement/repository"
	"personal-finance/internal/domain/transaction/service"
	"personal-finance/internal/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	mockedUUID = uuid.New()

	mockedTime        = time.Date(2022, 9, 15, 0, 0, 0, 0, time.UTC)
	aluguelmockedTime = time.Date(2022, time.September, 0o1, 0, 0, 0, 0, time.Local)
	energiaMockedTime = time.Date(2022, time.September, 15, 0, 0, 0, 0, time.Local)
	aguarMockedTime   = time.Date(2022, time.September, 30, 0, 0, 0, 0, time.Local)

	transactionsMock = []model.Transaction{
		{
			TransactionID: &mockedUUID,
			Estimate: &model.Movement{
				ID:            &mockedUUID,
				Description:   "Aluguel",
				Amount:        1000.0,
				Date:          &aluguelmockedTime,
				StatusID:      2,
				WalletID:      1,
				TypePaymentID: 1,
				CategoryID:    2,
				DateCreate:    mockedTime,
				DateUpdate:    mockedTime,
			},
			Consolidation: &model.Consolidation{
				Estimated: 1000.0,
				Realized:  1000.0,
			},
			DoneList: model.MovementList{
				{
					Description:   "Aluguel",
					Amount:        1000.0,
					Date:          &aluguelmockedTime,
					WalletID:      1,
					TypePaymentID: 1,
					CategoryID:    2,
					DateCreate:    mockedTime,
					DateUpdate:    mockedTime,
				},
			},
		},
		{
			TransactionID: &mockedUUID,
			Estimate: &model.Movement{
				Description:   "Energia",
				Amount:        300.0,
				Date:          &energiaMockedTime,
				WalletID:      1,
				TypePaymentID: 1,
				CategoryID:    2,
				DateCreate:    mockedTime,
				DateUpdate:    mockedTime,
			},
			Consolidation: &model.Consolidation{
				Estimated: 300.0,
				Realized:  300.0,
			},
			DoneList: model.MovementList{
				{
					Description:   "Energia",
					Amount:        300.0,
					Date:          &energiaMockedTime,
					WalletID:      1,
					TypePaymentID: 1,
					CategoryID:    2,
					DateCreate:    mockedTime,
					DateUpdate:    mockedTime,
				},
			},
		},
		{
			TransactionID: &mockedUUID,
			Estimate:      &model.Movement{},
			Consolidation: &model.Consolidation{},
			DoneList: model.MovementList{
				{
					Description:   "Agua",
					Amount:        120.0,
					Date:          &aguarMockedTime,
					WalletID:      1,
					TypePaymentID: 1,
					CategoryID:    2,
					DateCreate:    mockedTime,
					DateUpdate:    mockedTime,
				},
			},
		},
	}
)

func TestTransaction_FindByID(t *testing.T) {
	type mocks struct {
		transaction model.Transaction
		err         error
		repoMock    func() *repository.Mock
	}
	tt := []struct {
		name                string
		inputID             uuid.UUID
		mocks               mocks
		expectedTransaction model.Transaction
		expectedErr         error
	}{
		{
			name:    "success",
			inputID: mockedUUID,
			mocks: mocks{
				transaction: model.Transaction{
					TransactionID: &mockedUUID,
					Estimate:      transactionsMock[0].Estimate,
					DoneList:      transactionsMock[0].DoneList,
				},
				err: nil,
				repoMock: func() *repository.Mock {
					repoMock := repository.Mock{}
					repoMock.On("FindByID", mockedUUID, "userID").
						Return(*transactionsMock[0].Estimate, nil)
					repoMock.On("FindByTransactionID", mockedUUID, 1, "userID").
						Return(transactionsMock[0].DoneList, nil)
					return &repoMock
				},
			},
			expectedTransaction: transactionsMock[0],
			expectedErr:         nil,
		},
		{
			name:    "findByID error",
			inputID: mockedUUID,
			mocks: mocks{
				transaction: model.Transaction{
					Estimate: transactionsMock[0].Estimate,
					DoneList: transactionsMock[0].DoneList,
				},
				err: nil,
				repoMock: func() *repository.Mock {
					repoMock := repository.Mock{}
					repoMock.On("FindByID", mockedUUID, "userID").
						Return(model.Movement{}, errors.New("repository error"))
					repoMock.On("FindByTransactionID", mockedUUID, 1, "userID").
						Return(transactionsMock[0].DoneList, nil)
					return &repoMock
				},
			},
			expectedTransaction: model.Transaction{},
			expectedErr:         fmt.Errorf("error to find estimate transactions: %w", errors.New("repository error")),
		},
		{
			name:    "findByTransactionID error",
			inputID: mockedUUID,
			mocks: mocks{
				transaction: model.Transaction{
					Estimate: transactionsMock[0].Estimate,
					DoneList: transactionsMock[0].DoneList,
				},
				err: nil,
				repoMock: func() *repository.Mock {
					repoMock := repository.Mock{}
					repoMock.On("FindByID", mockedUUID, "userID").
						Return(*transactionsMock[0].Estimate, nil)
					repoMock.On("FindByTransactionID", mockedUUID, 1, "userID").
						Return(model.MovementList{}, errors.New("repository error"))
					return &repoMock
				},
			},
			expectedTransaction: model.Transaction{},
			expectedErr:         fmt.Errorf("error to find done transactions: %w", errors.New("repository error")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := tc.mocks.repoMock()

			svc := service.NewTransactionService(nil, repoMock)
			result, err := svc.FindByID(context.Background(), tc.inputID, "userID")

			require.Equal(t, tc.expectedTransaction, result)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestTransaction_FindByPeriod(t *testing.T) {
	mockedUUIDAluguel := uuid.New()
	mockedUUIDEnergia := uuid.New()
	type mocks struct {
		err      error
		repoMock func() *repository.Mock
	}
	tt := []struct {
		name                string
		inputPeriod         model.Period
		mocks               mocks
		expectedTransaction []model.Transaction
		expectedErr         error
	}{
		{
			name:        "success",
			inputPeriod: mockPeriod(),
			mocks: mocks{
				err: nil,
				repoMock: func() *repository.Mock {
					repoMock := repository.Mock{}
					repoMock.On("FindByStatusByPeriod", 2, mockPeriod(), "userID").
						Return([]model.Movement{
							mockMovement(mockedUUIDAluguel, "Aluguel", 1000, 2),
							mockMovement(mockedUUIDEnergia, "Energia", 1000, 2),
						}, nil)
					repoMock.On("FindByTransactionID", mockedUUIDAluguel, 1, "userID").
						Return(model.MovementList{
							mockMovement(mockedUUID, "Aluguel", 1000, 1),
						}, nil)
					repoMock.On("FindByTransactionID", mockedUUIDEnergia, 1, "userID").
						Return(model.MovementList{
							mockMovement(mockedUUID, "Enegia", 1000, 1),
						}, nil)
					repoMock.On("FindSingleTransactionByPeriod", 1, mockPeriod()).
						Return([]model.Movement{}, nil)
					return &repoMock
				},
			},
			expectedTransaction: []model.Transaction{
				mockTransaction(mockedUUIDAluguel, mockMovement(mockedUUIDAluguel, "Aluguel", 1000, 2),
					model.Consolidation{
						Estimated: 1000,
						Realized:  1000,
					},
					model.MovementList{
						mockMovement(mockedUUID, "Aluguel", 1000, 1),
					}),
				mockTransaction(mockedUUIDEnergia, mockMovement(mockedUUIDEnergia, "Energia", 1000, 2),
					model.Consolidation{
						Estimated: 1000,
						Realized:  1000,
					},
					model.MovementList{
						mockMovement(mockedUUID, "Enegia", 1000, 1),
					}),
			},
			expectedErr: nil,
		},
		{
			name:        "success",
			inputPeriod: mockPeriod(),
			mocks: mocks{
				err: nil,
				repoMock: func() *repository.Mock {
					repoMock := repository.Mock{}
					repoMock.On("FindByStatusByPeriod", 2, mockPeriod(), "userID").
						Return([]model.Movement{
							mockMovement(mockedUUIDAluguel, "Aluguel", 1000, 2),
							mockMovement(mockedUUIDEnergia, "Energia", 1000, 2),
						}, nil)
					repoMock.On("FindByTransactionID", mockedUUIDAluguel, 1, "userID").
						Return(model.MovementList{
							mockMovement(mockedUUID, "Aluguel", 1000, 1),
						}, nil)
					repoMock.On("FindByTransactionID", mockedUUIDEnergia, 1, "userID").
						Return(model.MovementList{
							mockMovement(mockedUUID, "Enegia", 1000, 1),
						}, nil)
					repoMock.On("FindSingleTransactionByPeriod", 1, mockPeriod()).
						Return([]model.Movement{}, nil)
					return &repoMock
				},
			},
			expectedTransaction: []model.Transaction{
				mockTransaction(mockedUUIDAluguel, mockMovement(mockedUUIDAluguel, "Aluguel", 1000, 2),
					model.Consolidation{
						Estimated: 1000,
						Realized:  1000,
					},
					model.MovementList{
						mockMovement(mockedUUID, "Aluguel", 1000, 1),
					}),
				mockTransaction(mockedUUIDEnergia, mockMovement(mockedUUIDEnergia, "Energia", 1000, 2),
					model.Consolidation{
						Estimated: 1000,
						Realized:  1000,
					},
					model.MovementList{
						mockMovement(mockedUUID, "Enegia", 1000, 1),
					}),
			},
			expectedErr: nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := tc.mocks.repoMock()

			svc := service.NewTransactionService(nil, repoMock)
			result, err := svc.FindByPeriod(context.Background(), tc.inputPeriod, "userID")

			require.Equal(t, tc.expectedTransaction, result)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}

func mockTransaction(
	transactionID uuid.UUID,
	estimate model.Movement,
	consolidation model.Consolidation,
	doneList model.MovementList,
) model.Transaction {
	return model.Transaction{
		TransactionID: &transactionID,
		Estimate:      &estimate,
		Consolidation: &consolidation,
		DoneList:      doneList,
	}
}

func mockMovement(id uuid.UUID, description string, amount float64, statusID int) model.Movement {
	return model.Movement{
		ID:          &id,
		Description: description,
		Amount:      amount,
		StatusID:    statusID,
	}
}

func mockPeriod() model.Period {
	return model.Period{
		From: mockedTime,
		To:   mockedTime.AddDate(0, 3, 0),
	}
}
