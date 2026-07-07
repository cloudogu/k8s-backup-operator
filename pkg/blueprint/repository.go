package blueprint

import (
	"testing"
)

type BlueprintWithDogus struct {
	DisplayName string
	DogusAsJson string
}

type Repository interface {
	FindDogusOfBlueprint() (BlueprintWithDogus, error)
}

func NewRepo(t *testing.T) {}
