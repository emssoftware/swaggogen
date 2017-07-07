package main

import "strconv"

type ValidationMap map[string]string

func (this ValidationMap) IsRequired() bool {
	_, ok := this["required"]
	return ok
}

func (this ValidationMap) Equals() (string, bool) {
	eq, ok := this["eq"]
	if !ok {
		return eq, false
	}

	return eq, true
}

func (this ValidationMap) Length() float64 {
	l_, ok := this["len"]
	if !ok {
		return -1
	}

	l, err := strconv.ParseFloat(l_, 64)
	if err != nil {
		return -1
	}

	return l
}

func (this ValidationMap) Max() float64 {
	max_, okMin := this["max"]
	lte_, okGte := this["lte"]
	if !okMin && !okGte {
		return -1
	}

	// Let's make max take precedent.
	if okMin {
		max, err := strconv.ParseFloat(max_, 64)
		if err == nil {
			return max
		}
	}

	if okGte {
		lte, err := strconv.ParseFloat(lte_, 64)
		if err == nil {
			return lte
		}
	}

	return -1
}

func (this ValidationMap) Min() float64 {
	min_, okMin := this["min"]
	gte_, okGte := this["gte"]
	if !okMin && !okGte {
		return -1
	}

	// Let's make min take precedent.
	if okMin {
		min, err := strconv.ParseFloat(min_, 64)
		if err == nil {
			return min
		}
	}

	if okGte {
		gte, err := strconv.ParseFloat(gte_, 64)
		if err == nil {
			return gte
		}
	}

	return -1
}

func (this ValidationMap) GreaterThan() float64 {
	gt_, ok := this["gt"]
	if !ok {
		return -1
	}

	gt, err := strconv.ParseFloat(gt_, 64)
	if err != nil {
		return -1.0
	}

	return gt
}

func (this ValidationMap) LessThan() float64 {
	lt_, ok := this["lt"]
	if !ok {
		return -1
	}

	lt, err := strconv.ParseFloat(lt_, 64)
	if err != nil {
		return -1
	}

	return lt
}
