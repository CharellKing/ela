package es

type V7Settings struct {
	*V6Settings
}

func NewV7Settings(settings, mappings map[string]interface{}, sourceIndex string) *V7Settings {
	return &V7Settings{
		V6Settings: NewV6Settings(settings, mappings, sourceIndex),
	}
}
