package multivalue

import (
	"fmt"
	"strconv"
)

type Vtype int
const (
	MVTYPE_UINT64          Vtype = iota
	MVTYPE_STRING
	MVTYPE_FLOAT64
	MVTYPE_EMPTY
)

type MultiValue struct {
	value  interface{}
	vtype  Vtype
}

// string
func NewString(s string) *MultiValue {
	return &MultiValue {
		value : s,
		vtype : MVTYPE_STRING,
	}
}

func (mv *MultiValue) IsString() bool {
	return mv.vtype == MVTYPE_STRING
}

func (mv *MultiValue) GetString() string {
	if mv.vtype != MVTYPE_STRING {
		return ""
	} else {
		return mv.value.(string)
	}
}

// uint64
func NewUint64(n uint64) *MultiValue {
	return &MultiValue {
		value : n,
		vtype : MVTYPE_UINT64,
	}
}

func (mv *MultiValue) IsUint64() bool {
	return mv.vtype == MVTYPE_UINT64
}

func (mv *MultiValue) GetUint64() uint64 {
	if mv.vtype != MVTYPE_UINT64 {
		return 0
	} else {
		return mv.value.(uint64)
	}
}

// float64
func NewFloat64(n float64) *MultiValue {
	return &MultiValue {
		value : n,
		vtype : MVTYPE_FLOAT64,
	}
}

func (mv *MultiValue) IsFloat64() bool {
	return mv.vtype == MVTYPE_FLOAT64
}

func (mv *MultiValue) GetFloat64() float64 {
	if mv.vtype != MVTYPE_FLOAT64 {
		return 0
	} else {
		return mv.value.(float64)
	}
}

// empty
func NewEmpty() *MultiValue {
	return &MultiValue {
		vtype : MVTYPE_EMPTY,
	}
}

func (mv *MultiValue) IsEmpty() bool {
	return mv.vtype == MVTYPE_EMPTY
}


// conversions
func (mv *MultiValue) AsString() string {
	if mv.vtype == MVTYPE_STRING {
		return mv.value.(string)
	} else if mv.vtype == MVTYPE_UINT64 {
		return fmt.Sprintf("%d", mv.value.(uint64))
	} else if mv.vtype == MVTYPE_FLOAT64 {
		return fmt.Sprintf("%f", mv.value.(float64))
	} else if mv.vtype == MVTYPE_EMPTY {
		return ""
	} else {
		return "<unknown_type>"
	}
}

func (mv *MultiValue) AsUint64() uint64 {
	if mv.vtype == MVTYPE_STRING {
		v, _ := strconv.ParseUint(mv.value.(string), 10, 64)
		return v
	} else if mv.vtype == MVTYPE_UINT64 {
		return mv.value.(uint64)
	} else if mv.vtype == MVTYPE_FLOAT64 {
		return uint64(mv.value.(float64))
	} else {
		return 0
	}
}

func (mv *MultiValue) AsFloat64() float64 {
	if mv.vtype == MVTYPE_STRING {
		v, _ := strconv.ParseFloat(mv.value.(string), 64)
		return v
	} else if mv.vtype == MVTYPE_UINT64 {
		return float64(mv.value.(uint64))
	} else if mv.vtype == MVTYPE_FLOAT64 {
		return mv.value.(float64)
	} else {
		return 0
	}
}

