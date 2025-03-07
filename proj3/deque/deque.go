package deque

import (
	"proj3/png"
	"sync/atomic"
	"unsafe"
)

type node struct {
	task *png.ImageTask // A pointer to an ImageTask payload.
	next unsafe.Pointer // unsafe.Pointer is used for lock-free operations.
}

type Deque struct {
	head unsafe.Pointer
	tail unsafe.Pointer
}

// NewDeque creates a new work-stealing deque
func NewDeque() *Deque {
	dummy := &node{} // dummy node
	// return a pointer to a new Deque with its head and tail pointers both pointing to the dummy node
	return &Deque{
		head: unsafe.Pointer(dummy),
		tail: unsafe.Pointer(dummy),
	}
}

// Push adds a task to the owner's end (LIFO)
func (d *Deque) Push(task *png.ImageTask) {
	newNode := &node{task: task}
	for {
		tail := (*node)(atomic.LoadPointer(&d.tail))
		next := (*node)(atomic.LoadPointer(&tail.next))

		// check if tail is still consistent
		if tail == (*node)(atomic.LoadPointer(&d.tail)) {
			// check if the current tail node's next pointer is nil, indicating that it's currently the actual tail (normal case)
			if next == nil {
				// atomically attempt to set the next pointer of the current tail node to the new node, if successful (no other thread modified it), it proceeds
				if atomic.CompareAndSwapPointer(&tail.next, nil, unsafe.Pointer(newNode)) {
					// atomically attempt to update the Deque's tail pointer to the newly added node
					atomic.CompareAndSwapPointer(&d.tail, unsafe.Pointer(tail), unsafe.Pointer(newNode))
					return
				}
			} else { // if next is not nil (another thread is in the process of adding a node), it helps the other thread by advancing the tail pointer
				atomic.CompareAndSwapPointer(&d.tail, unsafe.Pointer(tail), unsafe.Pointer(next))
			}
		}
	}
}

// Pop removes a task from the owner's end (LIFO)
func (d *Deque) Pop() (*png.ImageTask, bool) {
	for {
		tail := (*node)(atomic.LoadPointer(&d.tail))
		head := (*node)(atomic.LoadPointer(&d.head))

		// if the deque is empty, return nil for the task and false to indicate that no task was removed
		if tail == head { // Empty or only dummy node
			return nil, false
		}

		// call the findPrevious helper function to find the node before the current tail node
		prev := d.findPrevious(tail)

		// check if findPrevious returned nil, indicating that the deque has been concurrently modified
		// if there was concurrent modification, restart the loop
		if prev == nil {
			continue
		}

		// atomically attempt to update the Deque's tail pointer to the prev node, effectively removing the current tail node
		if atomic.CompareAndSwapPointer(&d.tail, unsafe.Pointer(tail), unsafe.Pointer(prev)) {
			return tail.task, true
		}
	}
}

// Steal removes a task from the victim's end (FIFO)
func (d *Deque) Steal() (*png.ImageTask, bool) {
	for {
		head := (*node)(atomic.LoadPointer(&d.head))
		tail := (*node)(atomic.LoadPointer(&d.tail))
		next := (*node)(atomic.LoadPointer(&head.next))

		// check if the head pointer is still consistent (hasn't been changed by another thread)
		// if there was concurrent modification, restart the loop
		if head != (*node)(atomic.LoadPointer(&d.head)) {
			continue // Concurrent modification
		}

		// if the deque is empty, return nil for the task and false to indicate that no task was stolen
		if head == tail { // Empty
			return nil, false
		}

		// atomically attempt to update the Deque's head pointer to the next node, effectively removing the current head node
		if atomic.CompareAndSwapPointer(&d.head, unsafe.Pointer(head), unsafe.Pointer(next)) {
			return next.task, true
		}
	}
}

// Helper to find previous node
func (d *Deque) findPrevious(target *node) *node {
	current := (*node)(atomic.LoadPointer(&d.head))
	for {
		next := (*node)(atomic.LoadPointer(&current.next))

		// if the next node is the target, it has found the previous node, so it returns the current node
		if next == target {
			return current
		}

		// check if the next node is nil, indicating the end of the deque.
		// if it reaches the end of the deque without finding the target, it returns nil
		if next == nil {
			return nil
		}

		// move to the next node in the deque.
		current = next
	}
}

// IsEmpty checks if the deque is empty
func (d *Deque) IsEmpty() bool {
	head := (*node)(atomic.LoadPointer(&d.head))
	tail := (*node)(atomic.LoadPointer(&d.tail))
	return head == tail
}
