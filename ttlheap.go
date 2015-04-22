package ttlru

type ttlHeap []*entry

func (h ttlHeap) Len() int {
	return len(h)
}

func (h ttlHeap) Less(i, j int) bool {
	return h[i].expires.Before(h[j].expires)
}

func (h ttlHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index, h[j].index = i, j
}

func (h *ttlHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*entry)
	item.index = n
	*h = append(*h, item)
}

func (h *ttlHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*h = old[0 : n-1]
	return item
}
