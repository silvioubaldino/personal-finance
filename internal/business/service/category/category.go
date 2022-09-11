package category

import (
	"context"
	"fmt"

	"personal-finance/internal/business/model"
	"personal-finance/internal/repositories/categories"
)

type Service interface {
	Add(ctx context.Context, car model.Category) (model.Category, error)
	FindAll(ctx context.Context) ([]model.Category, error)
	FindByID(ctx context.Context, ID int) (model.Category, error)
	Update(ctx context.Context, ID int, car model.Category) (model.Category, error)
	Delete(ctx context.Context, ID int) error
}

type service struct {
	repo categories.Repository
}

func NewService(repo categories.Repository) Service {
	return service{
		repo: repo,
	}
}

func (s service) Add(ctx context.Context, category model.Category) (model.Category, error) {
	result, err := s.repo.Add(ctx, category)
	if err != nil {
		return model.Category{}, fmt.Errorf("error to add categories: %w", err)
	}
	return result, nil
}

func (s service) FindAll(ctx context.Context) ([]model.Category, error) {
	resultList, err := s.repo.FindAll(ctx)
	if err != nil {
		return []model.Category{}, fmt.Errorf("error to find categories: %w", err)
	}
	return resultList, nil
}

func (s service) FindByID(ctx context.Context, id int) (model.Category, error) {
	result, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return model.Category{}, fmt.Errorf("error to find categories: %w", err)
	}
	return result, nil
}

func (s service) Update(ctx context.Context, id int, car model.Category) (model.Category, error) {
	result, err := s.repo.Update(ctx, id, car)
	if err != nil {
		return model.Category{}, fmt.Errorf("error updating categories: %w", err)
	}
	return result, nil
}

func (s service) Delete(ctx context.Context, id int) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting categories: %w", err)
	}
	return nil
}

/*func (s service) FindAll(ctx context.Context) ([]model.Car, error) {
	carList, err := s.repo.FindAll(ctx)
	if err != nil {
		return []model.Car{}, fmt.Errorf("error getting cars: %w", err)
	}
	return carList, nil
}

func (s service) FindByID(ctx context.Context, id string) (model.Car, error) {
	car, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return model.Car{}, fmt.Errorf("error getting categories by id: %w", err)
	}
	return car, nil
}

func (s service) Update(ctx context.Context, id string, car model.Car) (model.Car, error) {
	if !isModelValid(&car) {
		return model.Car{}, model.BuildBusinessError(
			"error updating categories: ",
			http.StatusUnprocessableEntity,
			model.ErrValidation,
		)
	}
	car, err := s.repo.Update(ctx, id, car)
	if err != nil {
		return model.Car{}, fmt.Errorf("error updating categories: %w", err)
	}
	return car, nil
}

func (s service) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting categories: %w", err)
	}
	return nil
}

func isModelValid(car *model.Car) bool {
	return !isNilOrEmpty(car.Model)
}

func isNilOrEmpty(x interface{}) bool {
	return x == "" || x == 0 || x == nil
}
*/
