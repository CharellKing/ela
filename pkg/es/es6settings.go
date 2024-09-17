package es

type V6Settings struct {
	*V5Settings
}

func NewV6Settings(settings, mappings, aliases map[string]interface{}, sourceIndex string) *V6Settings {
	return &V6Settings{
		V5Settings: NewV5Settings(settings, mappings, aliases, sourceIndex),
	}
}
