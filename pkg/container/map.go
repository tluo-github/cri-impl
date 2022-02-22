package container

import (
	"errors"
	"github.com/tluo-github/cri-impl/pkg/rollback"
)

type Map struct {
	byid   map[ID]*Container
	byname map[string]*Container
}

func NewMap() *Map {
	return &Map{
		byid:   make(map[ID]*Container),
		byname: make(map[string]*Container),
	}
}

func (m *Map) Add(c *Container, rb *rollback.Rollback) error {
	if _, ok := m.byid[c.ID()]; ok {
		return errors.New("Container ID exist")
	}
	if _, ok := m.byname[c.Name()]; ok {
		return errors.New("Container ID exist")
	}

	m.byid[c.ID()] = c
	m.byname[c.Name()] = c

	if rb != nil {
		// 添加回滚函数,执行删除
		rb.Add(func() {
			m.Del(c.ID())
		})
	}
	return nil
}

func (m *Map) Get(id ID) *Container {
	c, _ := m.byid[id]
	return c
}

func (m *Map) GetByName(name string) *Container {
	c, _ := m.byname[name]
	return c
}

func (m *Map) All() (cs []*Container) {
	for _, c := range m.byid {
		cs = append(cs, c)
	}
	return
}

func (m *Map) Del(id ID) bool {
	c, ok := m.byid[id]
	if ok {
		delete(m.byid, id)
		delete(m.byname, c.Name())
	}
	return ok
}
