package bulkexportv1

import "github.com/google/uuid"

type UUIDRunIDProvider struct{}

func NewUUIDRunIDProvider() *UUIDRunIDProvider {
	return &UUIDRunIDProvider{}
}

func (p *UUIDRunIDProvider) NewRunID() string {
	return uuid.New().String()
}
