package _import

import (
	"context"
	"errors"
	"fmt"
	"io"
	movementService "personal-finance/internal/domain/movement/service"
	"strconv"
	"strings"
	"time"

	categoryService "personal-finance/internal/domain/category/service"
	walletService "personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/filereader"
)

const (
	_uplannerIndexDate        = 0
	_uplannerIndexType        = 1
	_uplannerIndexAmount      = 2
	_uplannerIndexCategory    = 3
	_uplannerIndexSubCategory = 4
	_uplannerIndexStatus      = 5
	_uplannerIndexWallet      = 6
	_uplannerIndexCreditCard  = 7
	_uplannerIndexInvoice     = 8
	_uplannerIndexDescription = 9
)

type Uplanner interface {
	Import(file io.ReadCloser, userID string) error
}

type errParseCSV struct {
	line int
	err  error
}

type uplanner struct {
	categoryService categoryService.Service
	walletService   walletService.Service
	movementService movementService.Movement
	errorList       []error
}

func NewUplanner(categoryService categoryService.Service, walletService walletService.Service, movementService movementService.Movement) Uplanner {
	return uplanner{
		categoryService: categoryService,
		walletService:   walletService,
		movementService: movementService,
	}
}

func (u uplanner) Import(file io.ReadCloser, userID string) error {
	csv, err := filereader.ReadCSV(file)
	if err != nil {
		return err
	}

	movements := u.parseCSV(csv, userID)
	if len(u.errorList) > 0 {
		fmt.Println("Errors found:")
		for _, err := range u.errorList {
			fmt.Println(err.Error())
		}
		defer func() { u.errorList = nil }()
		return errors.New("errors found")
	}

	for _, movement := range movements {
		_, err := u.movementService.Add(context.Background(), movement, userID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (u *uplanner) parseCSV(csv [][]string, userID string) []model.Movement {
	var movements []model.Movement
	categories, err := u.categoryService.FindAll(context.Background(), userID)
	if err != nil {
		return nil
	}
	wallets, err := u.walletService.FindAll(context.Background(), userID)
	if err != nil {
		return nil
	}

	csv = csv[1:]
	for i, line := range csv {
		date, err := parseDate(line[_uplannerIndexDate])
		if err != nil {
			u.errorList = append(u.errorList, fmt.Errorf("line %d: %s", i+2, err.Error()))
		}

		amount, err := parseAmount(line[_uplannerIndexAmount])
		if err != nil {
			u.errorList = append(u.errorList, fmt.Errorf("line %d: %s", i+2, err.Error()))
		}

		categoryID, subCategoryID, err := findCategory(categories, line[_uplannerIndexCategory], line[_uplannerIndexSubCategory])
		if err != nil {
			u.errorList = append(u.errorList, fmt.Errorf("line %d: %s", i+2, err.Error()))
		}

		status, err := getStatus(line[_uplannerIndexStatus])
		if err != nil {
			u.errorList = append(u.errorList, fmt.Errorf("line %d: %s", i+2, err.Error()))
		}

		walletID, err := findWallet(wallets, line[_uplannerIndexWallet])
		if err != nil {
			u.errorList = append(u.errorList, fmt.Errorf("line %d: %s", i+2, err.Error()))
		}

		movement := model.Movement{
			Date:          &date,
			Amount:        amount,
			CategoryID:    categoryID,
			SubCategoryID: subCategoryID,
			StatusID:      status,
			WalletID:      walletID,
			Description:   line[_uplannerIndexDescription],
			TypePaymentID: 2,
			UserID:        userID,
		}

		movements = append(movements, movement)
	}

	return movements
}

func findCategory(categories []model.Category, category string, subCategory string) (categoryID int, subCategoryID int, err error) {
	var foundCategoryID int
	var foundSubCategoryID int
	for _, c := range categories {
		if c.Description == category {
			foundCategoryID = c.ID

			for _, sc := range c.SubCategories {
				if sc.Description == subCategory {
					foundSubCategoryID = sc.ID
					return foundCategoryID, foundSubCategoryID, nil
				}
			}
			return foundCategoryID, 0, errors.New("sub category not found")
		}
	}
	return 0, 0, errors.New("category not found")
}

func findWallet(wallets []model.Wallet, wallet string) (int, error) {
	for _, w := range wallets {
		if w.Description == wallet {
			return w.ID, nil
		}
	}
	return 0, errors.New("wallet not found")
}

func parseDate(stringDate string) (time.Time, error) {
	replacedDate := strings.Replace(stringDate, "\\xc3\\xaf\\xc2\\xbb\\xc2\\xbf", "", -1) //ï»¿
	date, err := time.Parse("02/01/2006", replacedDate)
	if err != nil {
		return time.Time{}, err
	}
	return date, nil
}

func getStatus(status string) (int, error) {
	if status == "Paga" || status == "Recebida" || status == "Transferida" {
		return model.TransactionStatusPaidID, nil
	}
	if status == "Não paga" || status == "Não recebida" || status == "Não transferida" {
		return model.TransactionStatusPlannedID, nil
	}
	return 0, errors.New("status not found")
}

func parseAmount(amount string) (float64, error) {
	amount = strings.Replace(amount, "R$", "", -1)
	amount = strings.Replace(amount, ",", ".", -1)
	amount = strings.TrimSpace(amount)
	return strconv.ParseFloat(amount, 64)
}
