package datatype

import (
	"container/heap"
	"fmt"
)

// An Item is something we manage in a priority queue.
type Item struct {
	value    interface{} // The value of the item; arbitrary.
	priority int         // The priority of the item in the queue.
	Index    int         // The index of the item in the heap.
}

func (i *Item) GetValue()(interface{}){
	return i.value
}
func (i *Item) GetPriority()(int){
	return i.priority
}
func (i *Item) SetValue(value interface{}){
	i.value = value
}
func (i *Item) SetPriority(priority int){
	i.priority = priority
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
func (pq *PriorityQueue) Update(item *Item, value interface{}, priority int) {
	item.value = value
	item.priority = priority
	heap.Fix(pq, item.Index)
}

func (pq *PriorityQueue) SetNodeValue(item *Item, value interface{}) {
	item.value = value
}

func (pq *PriorityQueue) PrintQueue(){
	t := make(PriorityQueue,pq.Len(),pq.Len())
	copy(t, *pq)
	for t.Len() > 0 {
		item := heap.Pop(&t)
		value := item.(*Item).GetValue()
		fmt.Println(value,item.(*Item).priority)
	}
}