package gosudachi

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
)

type SettingsJSON struct {
	BaseConfig
	path                     string
	inputTextPlugin          []json.RawMessage
	oovProviderPlugin        []json.RawMessage
	pathRewritePlugin        []json.RawMessage
	editConnectionCostPlugin []json.RawMessage
}

func NewSettingsJSON() *SettingsJSON {
	return &SettingsJSON{}
}

func (settings *SettingsJSON) GetBaseConfig() *BaseConfig {
	return &settings.BaseConfig
}

func (settings *SettingsJSON) ParseSettingsJSON(defpath string, reader io.Reader) error {
	internalBaseConfig := &struct {
		Path                     *string
		SystemDict               *string
		CharacterDefinitionFile  *string
		Utf16String              *bool
		UserDict                 *[]string
		InputTextPlugin          *[]json.RawMessage
		OovProviderPlugin        *[]json.RawMessage
		PathRewritePlugin        *[]json.RawMessage
		EditConnectionCostPlugin *[]json.RawMessage
	}{}

	decoder := json.NewDecoder(reader)
	err := decoder.Decode(internalBaseConfig)
	if err != nil {
		return err
	}
	if internalBaseConfig.Path == nil {
		settings.path = defpath
	} else {
		settings.path = *internalBaseConfig.Path
	}
	if internalBaseConfig.SystemDict != nil {
		settings.SystemDict = settings.getPath(*internalBaseConfig.SystemDict)
	}
	if internalBaseConfig.CharacterDefinitionFile != nil {
		settings.CharacterDefinitionFile = settings.getPath(*internalBaseConfig.CharacterDefinitionFile)
	}
	if internalBaseConfig.Utf16String != nil {
		settings.Utf16String = *internalBaseConfig.Utf16String
	}
	if internalBaseConfig.UserDict != nil {
		for _, ud := range *internalBaseConfig.UserDict {
			settings.UserDict = append(settings.UserDict, settings.getPath(ud))
		}
	}

	if internalBaseConfig.InputTextPlugin != nil {
		settings.inputTextPlugin = *internalBaseConfig.InputTextPlugin
	}
	if internalBaseConfig.OovProviderPlugin != nil {
		settings.oovProviderPlugin = *internalBaseConfig.OovProviderPlugin
	}
	if internalBaseConfig.PathRewritePlugin != nil {
		settings.pathRewritePlugin = *internalBaseConfig.PathRewritePlugin
	}
	if internalBaseConfig.EditConnectionCostPlugin != nil {
		settings.editConnectionCostPlugin = *internalBaseConfig.EditConnectionCostPlugin
	}
	return nil
}

func (settings *SettingsJSON) getPath(path string) string {
	if path == "" || filepath.IsAbs(path) || settings.path == "" {
		return path
	}
	return filepath.Join(settings.path, path)
}

func (settings *SettingsJSON) GetInputTextPluginArray(makeproc MakeInputTextPluginFunc) ([]InputTextPlugin, error) {
	ret := []InputTextPlugin{}
	pname := &struct {
		Class *string
		Name  *string
	}{}
	for _, raw := range settings.inputTextPlugin {
		err := json.Unmarshal(raw, pname)
		if err != nil {
			return ret, err
		}
		var name string
		if pname.Class != nil {
			name = *pname.Class
		}
		if pname.Name != nil {
			name = *pname.Name
		}
		plugin := makeproc(name)
		if plugin == nil {
			return ret, fmt.Errorf("InputTextPlugin: %s is unknown", name)
		}
		err = json.Unmarshal(raw, plugin.GetConfigStruct())
		if err != nil {
			return ret, err
		}
		ret = append(ret, plugin)
	}
	return ret, nil
}

func (settings *SettingsJSON) GetOovProviderPluginArray(makeproc MakeOovProviderPluginFunc) ([]OovProviderPlugin, error) {
	ret := []OovProviderPlugin{}
	pname := &struct {
		Class *string
		Name  *string
	}{}
	for _, raw := range settings.oovProviderPlugin {
		err := json.Unmarshal(raw, pname)
		if err != nil {
			return ret, err
		}
		var name string
		if pname.Class != nil {
			name = *pname.Class
		}
		if pname.Name != nil {
			name = *pname.Name
		}
		plugin := makeproc(name)
		if plugin == nil {
			return ret, fmt.Errorf("OovProviderPlugin: %s is unknown", name)
		}
		err = json.Unmarshal(raw, plugin.GetConfigStruct())
		if err != nil {
			return ret, err
		}
		ret = append(ret, plugin)
	}
	return ret, nil
}

func (settings *SettingsJSON) GetEditConnectionCostPluginArray(makeproc MakeEditConnectionCostPluginFunc) ([]EditConnectionCostPlugin, error) {
	ret := []EditConnectionCostPlugin{}
	pname := &struct {
		Class *string
		Name  *string
	}{}
	for _, raw := range settings.editConnectionCostPlugin {
		err := json.Unmarshal(raw, pname)
		if err != nil {
			return ret, err
		}
		var name string
		if pname.Class != nil {
			name = *pname.Class
		}
		if pname.Name != nil {
			name = *pname.Name
		}
		plugin := makeproc(name)
		if plugin == nil {
			return ret, fmt.Errorf("EditConnectionCostPlugin: %s is unknown", name)
		}
		err = json.Unmarshal(raw, plugin.GetConfigStruct())
		if err != nil {
			return ret, err
		}
		ret = append(ret, plugin)
	}
	return ret, nil
}

func (settings *SettingsJSON) GetPathRewritePluginArray(makeproc MakePathRewritePluginFunc) ([]PathRewritePlugin, error) {
	ret := []PathRewritePlugin{}
	pname := &struct {
		Class *string
		Name  *string
	}{}
	for _, raw := range settings.pathRewritePlugin {
		err := json.Unmarshal(raw, pname)
		if err != nil {
			return ret, err
		}
		var name string
		if pname.Class != nil {
			name = *pname.Class
		}
		if pname.Name != nil {
			name = *pname.Name
		}
		plugin := makeproc(name)
		if plugin == nil {
			return ret, fmt.Errorf("PathRewritePlugin: %s is unknown", name)
		}
		err = json.Unmarshal(raw, plugin.GetConfigStruct())
		if err != nil {
			return ret, err
		}
		ret = append(ret, plugin)
	}
	return ret, nil
}
