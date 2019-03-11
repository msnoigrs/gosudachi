package gosudachi

import (
	"fmt"
	"io"

	"github.com/msnoigrs/gosudachi/dictionary"
)

type JapaneseTokenizer struct {
	grammar            *dictionary.Grammar
	lexicon            *dictionary.LexiconSet
	inputTextPlugins   []InputTextPlugin
	oovProviderPlugins []OovProviderPlugin
	pathRewritePlugins []PathRewritePlugin
	defaultOovProvider OovProviderPlugin

	DumpOutput io.Writer
	lattice    *Lattice
}

func NewJapaneseTokenizer(grammar *dictionary.Grammar, lexicon *dictionary.LexiconSet, inputTextPlugins []InputTextPlugin, oovProviderPlugins []OovProviderPlugin, pathRewritePlugins []PathRewritePlugin) *JapaneseTokenizer {
	ret := &JapaneseTokenizer{
		grammar:            grammar,
		lexicon:            lexicon,
		inputTextPlugins:   inputTextPlugins,
		oovProviderPlugins: oovProviderPlugins,
		pathRewritePlugins: pathRewritePlugins,
		lattice:            NewLattice(grammar),
	}
	if len(oovProviderPlugins) > 0 {
		ret.defaultOovProvider = oovProviderPlugins[0]
	}
	return ret
}

func (t *JapaneseTokenizer) Tokenize(mode string, text string) (*MorphemeList, error) {
	inputTextBuilder := NewInputTextBuilder(text, t.grammar)

	if len(text) == 0 {
		return NewMorphemeList(inputTextBuilder.Build(), t.grammar, t.lexicon, []*LatticeNode{}), nil
	}

	for _, plugin := range t.inputTextPlugins {
		err := plugin.Rewrite(inputTextBuilder)
		if err != nil {
			return nil, err
		}
	}
	input := inputTextBuilder.Build()

	if t.DumpOutput != nil {
		fmt.Fprintln(t.DumpOutput, "=== Input dump")
		fmt.Fprintln(t.DumpOutput, input.GetText())
	}

	err := t.buildLattice(input)
	if err != nil {
		return nil, err
	}

	if t.DumpOutput != nil {
		fmt.Fprintln(t.DumpOutput, "=== Lattice dump")
		t.lattice.Dump(t.DumpOutput)
	}

	path, err := t.lattice.GetBestPath()
	if err != nil {
		return nil, err
	}

	if t.DumpOutput != nil {
		fmt.Fprintln(t.DumpOutput, "=== Before rewriting:")
		t.dumpPath(path)
	}

	for _, plugin := range t.pathRewritePlugins {
		err := plugin.Rewrite(input, &path, t.lattice)
		if err != nil {
			return nil, err
		}
	}
	t.lattice.clear()

	if mode != "C" {
		path, err = t.splitPath(path, mode)
		if err != nil {
			return nil, err
		}
	}

	if t.DumpOutput != nil {
		fmt.Fprintln(t.DumpOutput, "=== After rewriting:")
		t.dumpPath(path)
		fmt.Fprintln(t.DumpOutput, "===")
	}

	return NewMorphemeList(input, t.grammar, t.lexicon, path), nil
}

func (t *JapaneseTokenizer) buildLattice(input *InputText) error {
	bytea := input.Bytea
	t.lattice.resize(len(bytea))
	for i, _ := range bytea {
		if !input.CanBow(i) || !t.lattice.HasPreviousNode(i) {
			continue
		}
		iterator := t.lexicon.Lookup(bytea, i)
		hasWords := iterator.Next()
		for iterator.Next() {
			wordId, end := iterator.Get()
			if err := iterator.Err(); err != nil {
				break
			}
			n := NewLatticeNode(
				t.lexicon,
				t.lexicon.GetLeftId(wordId),
				t.lexicon.GetRightId(wordId),
				t.lexicon.GetCost(wordId),
				wordId,
			)
			t.lattice.Insert(i, end, n)
		}
		if err := iterator.Err(); err != nil {
			return err
		}

		// OOV
		types := input.GetCharCategoryTypes(i)
		if (types & dictionary.NOOOVBOW) != dictionary.NOOOVBOW {
			for _, plugin := range t.oovProviderPlugins {
				nodes, err := GetOOV(plugin, input, i, hasWords)
				if err != nil {
					return err
				}
				for _, node := range nodes {
					hasWords = true
					t.lattice.Insert(node.Begin, node.End, node)
				}
			}
		}
		if !hasWords && t.defaultOovProvider != nil {
			nodes, err := GetOOV(t.defaultOovProvider, input, i, hasWords)
			if err != nil {
				return err
			}
			for _, node := range nodes {
				hasWords = true
				t.lattice.Insert(node.Begin, node.End, node)
			}
		}
		if !hasWords {
			return fmt.Errorf("there is no morpheme at %d", i)
		}
	}
	t.lattice.connectEosNode()

	return nil
}

func (t *JapaneseTokenizer) splitPath(path []*LatticeNode, mode string) ([]*LatticeNode, error) {
	newPath := []*LatticeNode{}
	for _, node := range path {
		wi, err := node.GetWordInfo()
		if err != nil {
			return newPath, err
		}
		var wids []int32
		if mode == "A" {
			wids = wi.AUnitSplit
		} else {
			wids = wi.BUnitSplit
		}
		if len(wids) == 0 || len(wids) == 1 {
			newPath = append(newPath, node)
		} else {
			offset := node.Begin
			for _, wid := range wids {
				n := NewLatticeNode(t.lexicon, 0, 0, 0, wid)
				n.Begin = offset
				nwi, err := n.GetWordInfo()
				if err != nil {
					return newPath, err
				}
				offset += int(nwi.HeadwordLength)
				n.End = offset
				newPath = append(newPath, n)
			}
		}
	}
	return newPath, nil
}

func (t *JapaneseTokenizer) dumpPath(path []*LatticeNode) {
	for i, node := range path {
		fmt.Fprintf(t.DumpOutput, "%d: %s\n", i, node.String())
	}
}
