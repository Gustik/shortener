package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testObj struct {
	Value string
}

func (o *testObj) Reset() {
	o.Value = ""
}

func TestPool_GetPut(t *testing.T) {
	p := New(func() *testObj { return &testObj{} })

	obj := p.Get()
	assert.NotNil(t, obj)

	obj.Value = "hello"
	p.Put(obj)
	assert.Equal(t, "", obj.Value, "Reset должен очистить состояние")
}

func TestPool_GetAfterPut(t *testing.T) {
	p := New(func() *testObj { return &testObj{} })

	obj := p.Get()
	obj.Value = "data"
	p.Put(obj)

	obj2 := p.Get()
	assert.Equal(t, "", obj2.Value, "объект из пула должен быть чистым")
}
