package repository

import "errors"

//TODO Вынести в общий lib вместе с остальными ошибками из других репо

var (
	// ErrNotFound возвращается, когда запрошенная сущность не найдена в БД
	ErrNotFound = errors.New("entity not found")

	// ErrAlreadyExists возвращается при попытке создать дубликат уникальной сущности
	ErrAlreadyExists = errors.New("entity already exists")

	// ErrInvalidInput возвращается при некорректных входных данных
	ErrInvalidInput = errors.New("invalid input data")
)
