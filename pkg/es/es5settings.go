package es

import (
	"fmt"
	"github.com/jinzhu/copier"
	path "github.com/segment-boneyard/go-map-path"
	"github.com/spf13/cast"
	"sort"
)

type V5Settings struct {
	Settings map[string]interface{}
	Mappings map[string]interface{}

	SourceIndex string
}

func NewV5Settings(settings, mappings map[string]interface{}, sourceIndex string) *V5Settings {
	return &V5Settings{
		Settings:    settings,
		Mappings:    mappings,
		SourceIndex: sourceIndex,
	}
}

func (v5 *V5Settings) getUnwrappedSettings(targetIndex string) map[string]interface{} {
	var copySourceSettings map[string]interface{}
	_ = copier.Copy(&copySourceSettings, v5.Settings)

	unwrappedSettingMap := cast.ToStringMap(path.Path(copySourceSettings, fmt.Sprintf("%s.settings.index", v5.SourceIndex)))
	unsupportedKey := []string{"provided_name", "creation_date", "uuid", "version"}
	for _, key := range unsupportedKey {
		delete(unwrappedSettingMap, key)
	}
	return unwrappedSettingMap
}

func (v5 *V5Settings) ToESV5Setting(targetIndex string) map[string]interface{} {
	return map[string]interface{}{
		"settings": map[string]interface{}{
			"index": v5.getUnwrappedSettings(targetIndex),
		},
	}
}

func (v5 *V5Settings) ToESV6Setting(targetIndex string) map[string]interface{} {
	return v5.ToESV5Setting(targetIndex)
}

func (v5 *V5Settings) ToESV7Setting(targetIndex string) map[string]interface{} {
	return v5.ToESV5Setting(targetIndex)
}

func (v5 *V5Settings) ToESV8Setting(targetIndex string) map[string]interface{} {
	unwrappedSetting := v5.getUnwrappedSettings(targetIndex)
	return map[string]interface{}{
		"settings": unwrappedSetting,
	}
}

func (v5 *V5Settings) getUnwrappedMappings() map[string]interface{} {
	var copySourceMappings map[string]interface{}
	_ = copier.Copy(&copySourceMappings, v5.Mappings)

	unwrappedMappings := cast.ToStringMap(path.Path(v5.Mappings, fmt.Sprintf("%s.mappings", v5.SourceIndex)))
	return unwrappedMappings
}

func (v5 *V5Settings) mergeUnWrappedMapping(unwrappedMappings map[string]interface{}) map[string]interface{} {
	var typePropertiesMapArray []map[string]interface{}
	for _, typeProperties := range unwrappedMappings {
		typePropertiesMap := cast.ToStringMap(typeProperties)
		if _, ok := typePropertiesMap["properties"]; !ok {
			continue
		}

		enabled := path.Path(typePropertiesMap, "_source.enabled")
		if enabled != nil && cast.ToBool(enabled) == false {
			continue
		}

		typePropertiesMapArray = append(typePropertiesMapArray, cast.ToStringMap(typePropertiesMap["properties"]))
	}

	sort.Slice(typePropertiesMapArray, func(i, j int) bool {
		return len(typePropertiesMapArray[i]) > len(typePropertiesMapArray[j])
	})

	mergedProperties := make(map[string]interface{})
	for _, typePropertiesMap := range typePropertiesMapArray {
		for key, value := range typePropertiesMap {
			mergedProperties[key] = value
		}
	}

	return map[string]interface{}{
		"properties": mergedProperties,
	}
}

func (v5 *V5Settings) ToESV5Mapping(targetIndex string) map[string]interface{} {
	unwrappedMappings := v5.getUnwrappedMappings()
	return map[string]interface{}{
		"mappings": unwrappedMappings,
	}
}

func (v5 *V5Settings) ToESV6Mapping(targetIndex string) map[string]interface{} {
	return v5.ToESV5Mapping(targetIndex)
}

func (v5 *V5Settings) ToESV7Mapping(targetIndex string) map[string]interface{} {
	return v5.ToESV5Mapping(targetIndex)
}

func (v5 *V5Settings) ToESV8Mapping() map[string]interface{} {
	unwrappedMappings := v5.getUnwrappedMappings()
	mergedProperties := v5.mergeUnWrappedMapping(unwrappedMappings)
	return map[string]interface{}{
		"mappings": mergedProperties,
	}
}

func (v5 *V5Settings) ToTargetV5Settings(targetIndex string) *V5Settings {
	return NewV5Settings(
		v5.ToESV5Setting(targetIndex),
		v5.ToESV5Mapping(targetIndex),
		targetIndex)
}

func (v5 *V5Settings) ToTargetV6Settings(targetIndex string) *V6Settings {
	return NewV6Settings(
		v5.ToESV6Setting(targetIndex),
		v5.ToESV6Mapping(targetIndex),
		targetIndex)
}

func (v5 *V5Settings) ToTargetV7Settings(targetIndex string) *V7Settings {
	return NewV7Settings(
		v5.ToESV7Setting(targetIndex),
		v5.ToESV7Mapping(targetIndex),
		targetIndex)
}

func (v5 *V5Settings) ToTargetV8Settings(targetIndex string) *V8Settings {
	return NewV8Settings(
		v5.ToESV8Setting(targetIndex),
		v5.ToESV8Mapping(),
		targetIndex)
}

func (v5 *V5Settings) GetIndex() string {
	return v5.SourceIndex
}

func (v5 *V5Settings) GetMappings() map[string]interface{} {
	return v5.Mappings
}

func (v5 *V5Settings) GetSettings() map[string]interface{} {
	return v5.Settings
}
