package gosudachi

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/msnoigrs/gosudachi/dictionary"
)

type LatticeNode struct {
	Begin            int
	End              int
	leftId           int16
	rightId          int16
	cost             int16
	wordId           int32
	totalCost        int
	bestPreviousNode *LatticeNode
	isConnectedToBOS bool
	isDefined        bool
	IsOov            bool
	extraWordInfo    *dictionary.WordInfo
	lexicon          *dictionary.LexiconSet
}

func NewLatticeNode(lexicon *dictionary.LexiconSet, leftId int16, rightId int16, cost int16, wordId int32) *LatticeNode {
	return &LatticeNode{
		lexicon:   lexicon,
		leftId:    leftId,
		rightId:   rightId,
		cost:      cost,
		wordId:    wordId,
		isDefined: true,
	}
}

func (ln *LatticeNode) SetParameter(leftId int16, rightId int16, cost int16) {
	ln.leftId = leftId
	ln.rightId = rightId
	ln.cost = cost
}

func (ln *LatticeNode) GetBegin() int {
	return ln.Begin
}

func (ln *LatticeNode) GetEnd() int {
	return ln.End
}

func (ln *LatticeNode) SetRange(begin int, end int) {
	ln.Begin = begin
	ln.End = end
}

func (ln *LatticeNode) IsOOV() bool {
	return ln.IsOov
}

func (ln *LatticeNode) SetOOV() {
	ln.IsOov = true
}

func (ln *LatticeNode) GetWordInfo() (*dictionary.WordInfo, error) {
	if !ln.isDefined {
		return nil, errors.New("this node has no WordInfo")
	}
	if ln.extraWordInfo != nil {
		return ln.extraWordInfo, nil
	}
	return ln.lexicon.GetWordInfo(ln.wordId), nil
}

func (ln *LatticeNode) SetWordInfo(wordInfo *dictionary.WordInfo) {
	ln.extraWordInfo = wordInfo
	ln.isDefined = true
}

func (ln *LatticeNode) GetPathCost() int {
	return int(ln.cost)
}

func (ln *LatticeNode) GetWordId() int {
	return int(uint32(ln.wordId))
}

func (ln *LatticeNode) GetDictionaryId() int {
	if !ln.isDefined || ln.extraWordInfo != nil {
		return -1
	}
	return ln.lexicon.GetDictionaryId(ln.wordId)
}

func (ln *LatticeNode) String() string {
	var (
		surface string
		pos     int16
	)

	if ln.isDefined {
		wi, err := ln.GetWordInfo()
		if err != nil {
			surface = fmt.Sprintf("%v", err)
			pos = -1
		} else {
			surface = wi.Surface
			pos = wi.PosId
		}
	} else {
		surface = "(null)"
		pos = -1
	}

	return fmt.Sprintf("%d %d %s(%d) %d %d %d %d", ln.Begin, ln.End, surface, ln.wordId, pos, ln.leftId, ln.rightId, ln.cost)
}

type Lattice struct {
	endLists  [][]*LatticeNode
	eosNode   *LatticeNode
	grammar   *dictionary.Grammar
	eosParams []int16
}

func NewLattice(grammar *dictionary.Grammar) *Lattice {
	bosNode := &LatticeNode{}
	bosParams := dictionary.BosParameter
	bosNode.SetParameter(bosParams[0], bosParams[1], bosParams[2])
	bosNode.isConnectedToBOS = true
	endLists := make([][]*LatticeNode, 1)
	singletonList := make([]*LatticeNode, 1)
	singletonList[0] = bosNode
	endLists[0] = singletonList
	return &Lattice{
		endLists:  endLists,
		grammar:   grammar,
		eosParams: dictionary.EosParameter,
	}
}

func (l *Lattice) resize(size int) {
	if size > len(l.endLists)-1 {
		l.expand(size)
	}
	l.eosNode = &LatticeNode{}
	l.eosNode.SetParameter(l.eosParams[0], l.eosParams[1], l.eosParams[2])
	l.eosNode.Begin = size
	l.eosNode.End = size
}

func (l *Lattice) clear() {
	for i := 1; i < len(l.endLists); i++ {
		l.endLists[i] = l.endLists[i][:0]
	}
}

func (l *Lattice) expand(newSize int) {
	reallen := newSize + 1
	oldlen := len(l.endLists)
	if oldlen < reallen {
		l.endLists = append(l.endLists, make([][]*LatticeNode, reallen-oldlen)...)
		for i := oldlen; i < reallen; i++ {
			l.endLists[i] = []*LatticeNode{}
		}
	}
}

func (l *Lattice) GetNodesWithEnd(end int) []*LatticeNode {
	return l.endLists[end]
}

func (l *Lattice) GetNodes(begin int, end int) []*LatticeNode {
	ret := make([]*LatticeNode, 0)
	for _, n := range l.endLists[end] {
		if n.Begin == begin {
			ret = append(ret, n)
		}
	}
	return ret
}

func (l *Lattice) GetMinimumNode(begin int, end int) *LatticeNode {
	var (
		ret     *LatticeNode
		mincost int16
	)
	for _, n := range l.endLists[end] {
		if n.Begin == begin {
			if ret == nil || mincost > n.cost {
				ret = n
				mincost = n.cost
			}
		}
	}
	return ret
}

func (l *Lattice) Insert(begin int, end int, node *LatticeNode) {
	l.endLists[end] = append(l.endLists[end], node)
	node.Begin = begin
	node.End = end

	l.connectNode(node)
}

func (l *Lattice) Remove(begin int, end int, node *LatticeNode) {
	t := l.endLists[end]
	for i, n := range t {
		if n == node {
			if len(t) > 1 {
				copy(t[i:], t[i+1:])
			}
			t[len(t)-1] = nil
			l.endLists[end] = t[:len(t)-1]
		}
	}
}

func (l *Lattice) HasPreviousNode(index int) bool {
	return len(l.endLists[index]) > 0
}

func (l *Lattice) connectNode(rNode *LatticeNode) {
	begin := rNode.Begin
	rNode.totalCost = math.MaxInt32
	for _, lNode := range l.endLists[begin] {
		if !lNode.isConnectedToBOS {
			continue
		}
		connectCost := l.grammar.GetConnectCost(lNode.rightId, rNode.leftId)
		if connectCost == dictionary.InhibitedConnection {
			continue // this connection is not allowed
		}
		cost := lNode.totalCost + int(connectCost)
		if cost < rNode.totalCost {
			rNode.totalCost = cost
			rNode.bestPreviousNode = lNode
		}
	}
	rNode.isConnectedToBOS = !(rNode.bestPreviousNode == nil)
	rNode.totalCost += int(rNode.cost)
}

func (l *Lattice) connectEosNode() {
	l.connectNode(l.eosNode)
}

func (l *Lattice) GetBestPath() ([]*LatticeNode, error) {
	if !l.eosNode.isConnectedToBOS { // EOS node
		return nil, errors.New("EOS isn't connected to BOS")
	}
	ret := make([]*LatticeNode, 0)
	for node := l.eosNode.bestPreviousNode; node != l.endLists[0][0]; node = node.bestPreviousNode {
		ret = append(ret, node)
	}

	if len(ret) > 2 {
		// reverse
		for i := len(ret)/2 - 1; i >= 0; i-- {
			opp := len(ret) - 1 - i
			ret[i], ret[opp] = ret[opp], ret[i]
		}
	}
	return ret, nil
}

func (l *Lattice) Dump(w io.Writer) {
	index := 0
	for i := len(l.endLists); i >= 0; i-- {
		var rNodes []*LatticeNode
		if i <= len(l.endLists)-1 {
			rNodes = l.endLists[i]
		} else {
			rNodes = []*LatticeNode{l.eosNode}
		}
		for _, rNode := range rNodes {
			var (
				surface, pos string
			)
			if !rNode.isDefined {
				surface = "(null)"
				pos = "BOS/EOS"
			} else {
				wi, err := rNode.GetWordInfo()
				if err != nil {
					surface = fmt.Sprintf("%v", err)
					pos = "(null)"
				} else {
					surface = wi.Surface
					posId := wi.PosId
					if posId < 0 {
						pos = "(null)"
					} else {
						pos = strings.Join(l.grammar.GetPartOfSpeechString(posId), ",")
					}
				}
			}

			fmt.Fprintf(w, "%d: %d %d %s(%d) %s %d %d %d: ", index, rNode.Begin, rNode.End, surface, rNode.wordId, pos, rNode.leftId, rNode.rightId, rNode.cost)
			index++

			for _, lNode := range l.endLists[rNode.Begin] {
				cost := l.grammar.GetConnectCost(lNode.rightId, rNode.leftId)
				fmt.Fprintf(w, "%d ", cost)
			}
			fmt.Fprintln(w, "")
		}
	}
}
