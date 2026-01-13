package pkg

import "github.com/google/uuid"

func SomeShi() string {
	return uuid.New().String()
}
