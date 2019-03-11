package gosudachi

import (
	"strings"
	"testing"
)

var s string = `
{
  "path" : "/usr/local/share/sudachi",
  "systemDict" : "system.dic",
  "characterDefinitionFile" : "char.def",
  "inputTextPlugin" : [
    { "class" : "com.worksap.nlp.sudachi.DefaultInputTextPlugin" },
    { "class" : "com.worksap.nlp.sudachi.ProlongedSoundMarkInputTextPlugin",
      "prolongedSoundMarks" : ["ー", "-", "⁓", "〜", "〰"],
      "replacementSymbol" : "ー"
    }
  ],
  "oovProviderPlugin" : [
    {
      "class" : "com.worksap.nlp.sudachi.MeCabOovProviderPlugin",
      "charDef" : "char.def",
      "unkDef" : "unk.def"
    },
    {
      "class" : "com.worksap.nlp.sudachi.SimpleOovProviderPlugin",
      "oovPOSStrings" : [ "補助記号", "一般", "*", "*", "*", "*" ],
      "leftId" : 5968,
      "rightId" : 5968,
      "cost" : 3857
    }
  ],
  "pathRewritePlugin" : [
    {
      "name" : "JoinNumericPlugin",
      "enableNormalize" : false
    },
    {
      "name" : "JoinKatakanaOovPlugin",
      "oovPOS" : [ "名詞", "普通名詞", "一般", "*", "*", "*" ],
      "minLength" : 3
    }
  ]
}
`

// TestSettingsJSON_ParseSettingsJSON
func TestSettingsJSON_ParseSettingsJSON(t *testing.T) {
	settings := NewSettingsJSON()
	err := settings.ParseSettingsJSON("", strings.NewReader(s))
	if err != nil {
		t.Errorf("fail to parse json: %s", err)
	}

	bc, err := settings.GetBaseConfig()
	want := "/usr/local/share/sudachi/system.dic"
	if bc.SystemDict != want {
		t.Errorf("invalid result. want = %s, got = %s", want, bc.SystemDict)
	}

	iplugins, err := settings.GetInputTextPluginArray(DefMakeInputTextPlugin)
	if err != nil {
		t.Errorf("GetInputTextPluginArray: %s", err)
	}
	if len(iplugins) != 2 {
		t.Errorf("invalid result. want = 2, got = %d", len(iplugins))
	}
}
