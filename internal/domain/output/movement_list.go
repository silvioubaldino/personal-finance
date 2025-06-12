package output

import (
	"personal-finance/internal/domain"
)

func ToMovementListOutput(inputs []domain.Movement) MovementListOutput {
	output := make(MovementListOutput, len(inputs))
	for i, movement := range inputs {
		output[i] = *ToMovementOutput(movement)
	}
	return output
}
