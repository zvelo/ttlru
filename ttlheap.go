package ttlru

type ttlHeap[K comparable, V any] []*entry[K, V]

func (h ttlHeap[K, V]) Len() int {
	return len(h)
}

func (h ttlHeap[K, V]) Less(i, j int) bool {
	if i == j || i < 0 || j < 0 {
		return false
	}
	return h[i].expires.Before(h[j].expires)
}

func (h ttlHeap[K, V]) Swap(i, j int) {
	if i == j || i < 0 || j < 0 {
		return
	}
	h[i], h[j] = h[j], h[i]
	h[i].index, h[j].index = i, j
}

func (h *ttlHeap[K, V]) Push(x interface{}) {
	n := len(*h)
	item := x.(*entry[K, V])
	item.index = n
	*h = append(*h, item)
}

func (h *ttlHeap[K, V]) Pop() interface{} {
	old := *h
	n := len(old)
	if n == 0 {
		return nil
	}
	item := old[n-1]
	item.index = -1 // for safety
	*h = old[0 : n-1]
	return item
}
