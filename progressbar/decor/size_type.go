package decor

import (
	"fmt"
	"io"
	"math"
	"strconv"
)

//go:generate stringer -type=SizeB1024 -trimprefix=_i
//go:generate stringer -type=SizeB1000 -trimprefix=_

const (
	_ib   SizeB1024 = iota + 1
	_iKiB SizeB1024 = 1 << (iota * 10)
	_iMiB
	_iGiB
	_iTiB
)

// SizeB1024 named type, which implements fmt.Formatter interface. It
// adjusts its value according to byte size multiple by 1024 and appends
// appropriate size marker (KiB, MiB, GiB, TiB).
type SizeB1024 int64

func (self SizeB1024) Format(st fmt.State, verb rune) {
	var prec int
	switch verb {
	case 'd':
	case 's':
		prec = -1
	default:
		if p, ok := st.Precision(); ok {
			prec = p
		} else {
			prec = 6
		}
	}

	var unit SizeB1024
	switch {
	case self < _iKiB:
		unit = _ib
	case self < _iMiB:
		unit = _iKiB
	case self < _iGiB:
		unit = _iMiB
	case self < _iTiB:
		unit = _iGiB
	case self <= math.MaxInt64:
		unit = _iTiB
	}

	io.WriteString(st, strconv.FormatFloat(float64(self)/float64(unit), 'f', prec, 64))

	if st.Flag(' ') {
		io.WriteString(st, " ")
	}
	io.WriteString(st, unit.String())
}

const (
	_b  SizeB1000 = 1
	_KB           = _b * 1000
	_MB           = _KB * 1000
	_GB           = _MB * 1000
	_TB           = _GB * 1000
)

// SizeB1000 named type, which implements fmt.Formatter interface. It
// adjusts its value according to byte size multiple by 1000 and appends
// appropriate size marker (KB, MB, GB, TB).
type SizeB1000 int64

func (self SizeB1000) Format(st fmt.State, verb rune) {
	var prec int
	switch verb {
	case 'd':
	case 's':
		prec = -1
	default:
		if p, ok := st.Precision(); ok {
			prec = p
		} else {
			prec = 6
		}
	}

	var unit SizeB1000
	switch {
	case self < _KB:
		unit = _b
	case self < _MB:
		unit = _KB
	case self < _GB:
		unit = _MB
	case self < _TB:
		unit = _GB
	case self <= math.MaxInt64:
		unit = _TB
	}

	io.WriteString(st, strconv.FormatFloat(float64(self)/float64(unit), 'f', prec, 64))

	if st.Flag(' ') {
		io.WriteString(st, " ")
	}
	io.WriteString(st, unit.String())
}