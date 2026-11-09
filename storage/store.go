package storage

import (
	"encoding/json"
	"errors"
	"os"
)

func NewStore(path string) *Store {
	return &Store{path: path}
}

// Load читает данные. Если файла нет или он пуст, возвращает пустые данные.
func (s *Store) Load() (Data, error) {
	data := Data{Requisites: map[int64]string{}}

	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return data, nil
		}
		return Data{}, err
	}
	if len(content) == 0 {
		return data, nil
	}
	if err := json.Unmarshal(content, &data); err != nil {
		return Data{}, err
	}
	if data.Requisites == nil {
		data.Requisites = map[int64]string{}
	}
	return data, nil
}

// Save атомарно перезаписывает файл: пишет во временный и переименовывает его.
func (s *Store) Save(data Data) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, content, 0600); err != nil {
		return err
	}
	return os.Rename(tempPath, s.path)
}
