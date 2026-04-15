package bulkexportv1

import "github.com/google/uuid"

type uuidRunIDProvider struct{}

func NewUUIDRunIDProvider() RunIDProvider {
	return &uuidRunIDProvider{}
}

func (p *uuidRunIDProvider) NewRunID() string {
	return uuid.New().String()
}
