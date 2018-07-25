package goconnpool

type deck struct {
	data []interface{}
}

func (d *deck) push(x interface{}) {
	d.data = append(d.data, x)
}

func (d *deck) pop() interface{} {
	if len(d.data) == 0 {
		return nil
	}

	x := d.data[0]
	d.data = d.data[1:]
	return x
}

func (d *deck) size() int {
	return len(d.data)
}

type roundRobin struct {
	idx  int
	data []interface{}
}

func (rr *roundRobin) push(x interface{}) {
	rr.data = append(rr.data, x)
}

func (rr *roundRobin) next() interface{} {
	x := rr.data[rr.idx]

	rr.idx++
	if rr.idx >= len(rr.data) {
		rr.idx = 0
	}

	return x
}

func (rr *roundRobin) size() int {
	return len(rr.data)
}
