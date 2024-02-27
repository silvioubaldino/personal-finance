package filereader

import (
	"encoding/csv"
	"io"
)

func ReadCSV(file io.ReadCloser) ([][]string, error) {
	reader := csv.NewReader(file)
	reader.Comma = ';'
	read, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	return read, err
}
