package gosudachi

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/msnoigrs/gosudachi/data"
	"github.com/msnoigrs/gosudachi/internal/lnreader"
	"golang.org/x/text/unicode/norm"
)

type DefaultInputTextPluginConfig struct {
	RewriteDef string
}

type DefaultInputTextPlugin struct {
	config             *DefaultInputTextPluginConfig
	rewriteDef         string
	ignoreNormalizeMap map[rune]bool
	keyLengths         map[rune]int
	replaceCharMap     map[string][]rune
}

func NewDefaultInputTextPlugin(config *DefaultInputTextPluginConfig) *DefaultInputTextPlugin {
	if config == nil {
		config = &DefaultInputTextPluginConfig{}
	}
	return &DefaultInputTextPlugin{
		config:             config,
		ignoreNormalizeMap: map[rune]bool{},
		keyLengths:         map[rune]int{},
		replaceCharMap:     map[string][]rune{},
	}
}

func (p *DefaultInputTextPlugin) GetConfigStruct() interface{} {
	if p.config == nil {
		p.config = &DefaultInputTextPluginConfig{}
	}
	return p.config
}

func (p *DefaultInputTextPlugin) SetUp() error {
	if p.rewriteDef == "" {
		p.rewriteDef = p.config.RewriteDef
	}
	p.config = nil
	if p.ignoreNormalizeMap == nil {
		p.ignoreNormalizeMap = map[rune]bool{}
	}
	if p.keyLengths == nil {
		p.keyLengths = map[rune]int{}
	}
	if p.replaceCharMap == nil {
		p.replaceCharMap = map[string][]rune{}
	}
	err := p.readRewriteLists(p.rewriteDef)
	if err != nil {
		return fmt.Errorf("DefaultInputTextPlugin: %s", err)
	}
	return nil
}

func (p *DefaultInputTextPlugin) getKeyLength(key rune, def int) int {
	l, ok := p.keyLengths[key]
	if !ok {
		return def
	}
	return l
}

func (p *DefaultInputTextPlugin) Rewrite(builder *InputTextBuilder) error {
	runes := builder.GetText()
	runelen := len(runes)

	utf8buf := make([]byte, 8, 8)

	offset := 0
	nextOffset := 0
TEXTLOOP:
	for i := 0; i < runelen; i++ {
		offset += nextOffset
		nextOffset = 0
		// 1. replace char without normalize
		for l := minInt(p.getKeyLength(runes[i], 0), runelen-i); l > 0; l-- {
			replace, ok := p.replaceCharMap[string(runes[i:i+l])]
			if ok {
				builder.Replace(i+offset, i+l+offset, replace)
				nextOffset += len(replace) - l
				i += l - 1
				continue TEXTLOOP
			}
		}

		// 2. normalize
		original := runes[i]

		// 2-1. capital alphabet (not only latin but greek, cyrillic, etc) -> small
		lower := unicode.ToLower(original)
		var replace []rune
		_, ok := p.ignoreNormalizeMap[lower]
		if ok {
			if original == lower {
				continue
			}
			replace = []rune{lower}
		} else {
			// 2-2. normalize (except in ignoreNormalize)
			//    e.g. full-width alphabet -> half-width / ligature / etc.
			size := utf8.EncodeRune(utf8buf, lower)
			replace = []rune(string(norm.NFKC.Bytes(utf8buf[:size])))
		}
		nextOffset = len(replace) - 1
		if len(replace) != 1 || original != replace[0] {
			builder.Replace(i+offset, i+1+offset, replace)
		}
	}
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *DefaultInputTextPlugin) readRewriteLists(rewriteDef string) error {
	var rewriteDefReader io.Reader
	if rewriteDef != "" {
		rewriteDefFd, err := os.OpenFile(rewriteDef, os.O_RDONLY, 0644)
		if err != nil {
			return fmt.Errorf("DefaultInputTextPlugin: %s: %s", err, rewriteDef)
		}
		defer rewriteDefFd.Close()
		rewriteDefReader = rewriteDefFd
	} else {
		rewiteDefF, err := data.Assets.Open("rewrite.def")
		if err != nil {
			return fmt.Errorf("DefaultInputTextPlugin: %s: (data.Assets)rewrite.def", err)
		}
		defer rewiteDefF.Close()
		rewriteDefReader = rewiteDefF
	}

	r := lnreader.NewLineNumberReader(rewriteDefReader)
	for {
		line, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("DefaultInputTextPlugin: %s", err)
		}
		if lnreader.IsSkipLine(line) {
			continue
		}
		cols := strings.Fields(string(line))
		if len(cols) == 1 {
			// ignored normalize list
			key := []rune(cols[0])
			if len(key) != 1 {
				return fmt.Errorf("DefaultInputTextPlugin: %s is already defined at line %d", cols[0], r.NumLine)
			}
			p.ignoreNormalizeMap[key[0]] = true
		} else if len(cols) == 2 {
			// replace char list
			_, ok := p.replaceCharMap[cols[0]]
			if ok {
				return fmt.Errorf("DefaultInputTextPlugin: %s is already defined at line %d", cols[0], r.NumLine)
			}
			key := []rune(cols[0])
			if p.getKeyLength(key[0], -1) < len(key) {
				// store the longest key length
				p.keyLengths[key[0]] = len(key)
			}
			p.replaceCharMap[cols[0]] = []rune(cols[1])
		} else {
			return fmt.Errorf("DefaultInputTextPlugin: invalid format at line %d", r.NumLine)
		}
	}
	return nil
}
