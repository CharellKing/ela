package es

type V8Settings struct {
	*V7Settings
}

func NewV8Settings(settings, mappings map[string]interface{}, sourceIndex string) *V8Settings {
	return &V8Settings{
		V7Settings: NewV7Settings(settings, mappings, sourceIndex),
	}
}

func (v8 *V8Settings) GetProperties() map[string]interface{} {
	return v8.V7Settings.V6Settings.V5Settings.getUnwrappedMappings()
}
