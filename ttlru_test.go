package ttlru

import (
	"container/heap"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGeneral(t *testing.T) {
	l := New(128, WithTTL(2*time.Second))

	require.NotNil(t, l)
	require.Equal(t, 0, l.Len())
	require.Equal(t, 128, l.Cap())

	for i := 0; i < 128; i++ {
		require.False(t, l.Set(i, i))
	}

	require.Equal(t, 128, l.Len())
	require.Equal(t, 128, l.Cap())

	for i := 128; i < 256; i++ {
		require.True(t, l.Set(i, i))
	}

	require.Equal(t, 128, l.Len())
	require.Equal(t, 128, l.Cap())

	for _, k := range l.Keys() {
		v, ok := l.Get(k)
		require.True(t, ok)
		require.Equal(t, v, k)
	}

	for i := 0; i < 128; i++ {
		val, ok := l.Get(i)
		require.False(t, ok)
		require.Nil(t, val)
	}

	for i := 128; i < 256; i++ {
		val, ok := l.Get(i)
		require.True(t, ok)
		require.Equal(t, val, i)
	}

	for i := 128; i < 192; i++ {
		require.True(t, l.Del(i))
		val, ok := l.Get(i)
		require.False(t, ok)
		require.Nil(t, val)
	}

	done := make(chan interface{})

	time.AfterFunc(3*time.Second, func() {
		require.Equal(t, 0, l.Len())
		require.Equal(t, 128, l.Cap())

		require.Equal(t, 0, l.Len())
		require.Equal(t, 128, l.Cap())

		require.False(t, l.Set(0, 0))
		require.Equal(t, 1, l.Len())
		require.Equal(t, 128, l.Cap())

		l.Purge()
		require.Equal(t, 0, l.Len())
		require.Equal(t, 128, l.Cap())

		val, ok := l.Get(200)
		require.False(t, ok)
		require.Nil(t, val)

		done <- true
	})

	<-done
}

func TestAddReturnsProperly(t *testing.T) {
	l := New(1, WithTTL(2*time.Second))
	require.NotNil(t, l)
	require.Equal(t, 0, l.Len())
	require.Equal(t, 1, l.Cap())

	require.False(t, l.Set(1, 1))
	require.Equal(t, 1, l.Len())
	require.Equal(t, 1, l.Cap())

	require.True(t, l.Set(2, 2))
	require.Equal(t, 1, l.Len())
	require.Equal(t, 1, l.Cap())
}

func TestInvalidCreation(t *testing.T) {
	require.Nil(t, New(0, WithTTL(1)))
	require.Nil(t, New(-1, WithTTL(1)))
	require.Nil(t, New(1, WithTTL(-1)))
}

func TestSetShouldAlsoUpdate(t *testing.T) {
	l := New(1, WithTTL(2*time.Second))
	require.NotNil(t, l)
	require.Equal(t, 0, l.Len())
	require.Equal(t, 1, l.Cap())

	require.False(t, l.Set(1, 1))
	require.Equal(t, 1, l.Len())
	require.Equal(t, 1, l.Cap())

	v, ok := l.Get(1)
	require.True(t, ok)
	require.Equal(t, 1, v)

	require.False(t, l.Set(1, 2))
	require.Equal(t, 1, l.Len())
	require.Equal(t, 1, l.Cap())

	v, ok = l.Get(1)
	require.True(t, ok)
	require.Equal(t, 2, v)
}

func TestDeleteShouldReturnProperly(t *testing.T) {
	l := New(1, WithTTL(2*time.Second))
	require.NotNil(t, l)
	require.Equal(t, 0, l.Len())
	require.Equal(t, 1, l.Cap())

	require.False(t, l.Set(1, 1))
	require.Equal(t, 1, l.Len())
	require.Equal(t, 1, l.Cap())

	v, ok := l.Get(1)
	require.True(t, ok)
	require.Equal(t, 1, v)

	require.True(t, l.Del(1))
	require.False(t, l.Del(2))
}

func TestItemShouldExpireDespiteGet(t *testing.T) {
	l := New(1, WithTTL(300*time.Millisecond), WithoutReset())
	require.NotNil(t, l)
	require.False(t, l.Set(1, 1))

	done := make(chan interface{})
	time.AfterFunc(200*time.Millisecond, func() {
		v, ok := l.Get(1)
		require.True(t, ok)
		require.Equal(t, 1, v)
		done <- true
	})
	<-done

	time.AfterFunc(200*time.Millisecond, func() {
		v, ok := l.Get(1)
		require.False(t, ok)
		require.Nil(t, v)
		done <- true
	})
	<-done
}

func TestWithoutTTL(t *testing.T) {
	l := New(2)
	require.NotNil(t, l)

	require.False(t, l.Set(1, 1))
	v, ok := l.Get(1)
	require.True(t, ok)
	require.Equal(t, 1, v)

	require.False(t, l.Set(2, 2))
	v, ok = l.Get(2)
	require.True(t, ok)
	require.Equal(t, 2, v)

	require.True(t, l.Set(3, 3))
	v, ok = l.Get(3)
	require.True(t, ok)
	require.Equal(t, 3, v)

	v, ok = l.Get(1)
	require.False(t, ok)
	require.Nil(t, v)

	v, ok = l.Get(2)
	require.True(t, ok)
	require.Equal(t, 2, v)
}

func TestTTLAfterPurge(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	l := New(1, WithTTL(10*time.Millisecond))
	l.Set("bug", "foo")

	l.Purge()

	// the bug caused a panic here in a different goroutine, so it couldn't be
	// recovered in the test.
	// if the test completes successfully, then there was obviously no panic

	<-ctx.Done()
}

func TestPopEmptyHeap(t *testing.T) {
	var h ttlHeap
	heap.Push(&h, &entry{value: 1})
	heap.Pop(&h)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("recovered from panic: %+v", r)
		}
	}()

	heap.Pop(&h)
}
