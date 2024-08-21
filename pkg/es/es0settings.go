package es

type IESSettings interface {
	ToESV5Setting(targetIndex string) map[string]interface{}
	ToESV6Setting(targetIndex string) map[string]interface{}
	ToESV7Setting(targetIndex string) map[string]interface{}
	ToESV8Setting(targetIndex string) map[string]interface{}

	ToESV5Mapping(targetIndex string) map[string]interface{}
	ToESV6Mapping(targetIndex string) map[string]interface{}
	ToESV7Mapping(targetIndex string) map[string]interface{}
	ToESV8Mapping() map[string]interface{}

	ToTargetV5Settings(targetIndex string) *V5Settings
	ToTargetV6Settings(targetIndex string) *V6Settings
	ToTargetV7Settings(targetIndex string) *V7Settings
	ToTargetV8Settings(targetIndex string) *V8Settings

	GetIndex() string
	GetMappings() map[string]interface{}
	GetSettings() map[string]interface{}
	GetProperties() map[string]interface{}
}
