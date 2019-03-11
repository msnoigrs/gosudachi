package gosudachi

type errState int

const (
	errNone errState = iota
	errPoint
	errComma
	errOther
)

type numericParser struct {
	digitLength     int
	isFirstDigit    bool
	hasComma        bool
	hasHangingPoint bool
	errorState      errState
	total           *stringNumber
	subtotal        *stringNumber
	tmp             *stringNumber
}

func newNumericParser() *numericParser {
	return &numericParser{
		isFirstDigit: true,
		total:        newStringNumber(),
		subtotal:     newStringNumber(),
		tmp:          newStringNumber(),
	}
}

type stringNumber struct {
	significand []rune
	scale       int
	point       int
	IsAllZero   bool
}

func newStringNumber() *stringNumber {
	return &stringNumber{
		point:     -1,
		IsAllZero: true,
	}
}

func (n *stringNumber) clear() {
	n.significand = n.significand[:0]
	n.scale = 0
	n.point = -1
	n.IsAllZero = true
}

func (n *stringNumber) append(i int) {
	if i != 0 {
		n.IsAllZero = false
	}
	n.significand = append(n.significand, intToRune(i))
}

func (n *stringNumber) setScale(i int) {
	if len(n.significand) == 0 {
		n.significand = append(n.significand, '1')
	}
	n.scale += i
}

func (n *stringNumber) add(t *stringNumber) bool {
	if len(t.significand) == 0 {
		return true
	}

	if len(n.significand) == 0 {
		n.significand = append(n.significand, t.significand...)
		n.scale = t.scale
		n.point = t.point
		return true
	}

	l := t.intLength()
	if n.scale >= l {
		n.fillZero(n.scale - l)
		if t.point >= 0 {
			n.point = len(n.significand) + t.point
		}
		_ = t.String()
		n.significand = append(n.significand, t.significand...)
		n.scale = t.scale
		return true
	}

	return false
}

func (n *stringNumber) setPoint() bool {
	if n.scale == 0 && n.point < 0 {
		n.point = len(n.significand)
		return true
	}
	return false
}

func (n *stringNumber) intLength() int {
	n.normalizeScale()
	if n.point >= 0 {
		return n.point
	}
	return len(n.significand) + n.scale
}

func (n *stringNumber) isZero() bool {
	return len(n.significand) == 0
}

func (n *stringNumber) String() string {
	if len(n.significand) == 0 {
		return "0"
	}

	n.normalizeScale()
	if n.scale > 0 {
		n.fillZero(n.scale)
	} else if n.point >= 0 {
		if n.point == 0 {
			n.significand = append(n.significand, []rune{0, 0}...)
			copy(n.significand[2:], n.significand[:len(n.significand)-2])
			n.significand[0] = '0'
			n.significand[1] = '.'
		} else {
			n.significand = append(n.significand, rune(0))
			copy(n.significand[n.point+1:], n.significand[n.point:])
			n.significand[n.point] = '.'
		}
		i := len(n.significand) - 1
		j := 0
		for i >= 0 && n.significand[i] == '0' {
			i--
			j++
		}
		if n.significand[i] == '.' {
			i--
			j++
		}
		if j > 0 {
			n.significand = n.significand[:i+1]
		}
	}

	return string(n.significand)
}

func (n *stringNumber) normalizeScale() {
	if n.point >= 0 {
		nScale := len(n.significand) - n.point
		if nScale > n.scale {
			n.point += n.scale
			n.scale = 0
		} else {
			n.scale -= nScale
			n.point = -1
		}
	}
}

func (n *stringNumber) fillZero(length int) {
	for i := 0; i < length; i++ {
		n.significand = append(n.significand, '0')
	}
}

func intToRune(i int) rune {
	return rune(int32('0') + int32(i))
}

func (p *numericParser) clear() {
	p.digitLength = 0
	p.isFirstDigit = true
	p.hasComma = false
	p.hasHangingPoint = false
	p.errorState = errNone
	p.total.clear()
	p.subtotal.clear()
	p.tmp.clear()
}

func (p *numericParser) checkComma() bool {
	if p.isFirstDigit {
		return false
	} else if !p.hasComma {
		return p.digitLength <= 3 && !p.tmp.isZero() && !p.tmp.IsAllZero
	} else {
		return p.digitLength == 3
	}
}

func (p *numericParser) append(c rune) bool {
	if c == '.' {
		p.hasHangingPoint = true
		if p.isFirstDigit {
			p.errorState = errPoint
			return false
		} else if p.hasComma && !p.checkComma() {
			p.errorState = errComma
			return false

		} else if p.tmp.setPoint() {
			p.errorState = errPoint
			return false
		}
		p.hasComma = false
		return true
	} else if c == ',' {
		if !p.checkComma() {
			p.errorState = errComma
			return false
		}
		p.hasComma = true
		p.digitLength = 0
		return true
	}

	n, ok := runeToNumMap[c]
	if !ok {
		return false
	}
	if n < 0 && n >= -3 { // isSmallUnit
		p.tmp.setScale(-n)
		if !p.subtotal.add(p.tmp) {
			return false
		}
		p.tmp.clear()
		p.isFirstDigit = true
		p.digitLength = 0
		p.hasComma = false
	} else if n <= -4 { // isLargeUnit
		if !p.subtotal.add(p.tmp) || p.subtotal.isZero() {
			return false
		}
		p.subtotal.setScale(-n)
		if !p.total.add(p.subtotal) {
			return false
		}
		p.subtotal.clear()
		p.tmp.clear()
		p.isFirstDigit = true
		p.digitLength = 0
		p.hasComma = false
	} else {
		p.tmp.append(n)
		p.isFirstDigit = false
		p.digitLength++
		p.hasHangingPoint = false
	}

	return true
}

func (p *numericParser) done() bool {
	ret := p.subtotal.add(p.tmp) && p.total.add(p.subtotal)
	if p.hasHangingPoint {
		p.errorState = errPoint
		return false
	} else if p.hasComma && p.digitLength != 3 {
		p.errorState = errComma
		return false
	}
	return ret
}

func (p *numericParser) getNormalized() string {
	return p.total.String()
}

var runeToNumMap = map[rune]int{
	'0': 0,
	'1': 1,
	'2': 2,
	'3': 3,
	'4': 4,
	'5': 5,
	'6': 6,
	'7': 7,
	'8': 8,
	'9': 9,
	'〇': 0,
	'一': 1,
	'二': 2,
	'三': 3,
	'四': 4,
	'五': 5,
	'六': 6,
	'七': 7,
	'八': 8,
	'九': 9,
	'十': -1,
	'百': -2,
	'千': -3,
	'万': -4,
	'億': -8,
	'兆': -12,
}
