package store

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"cloud.google.com/go/firestore"
)

func MockStore() *StoreMock {
	return &StoreMock{
		rootNode: DocNode{
			colNodes: make(map[string]*ColNode),
		},
		failsOnGet: make(map[string]bool),
		failsOnSet: make(map[string]bool),
		failsOnDel: make(map[string]bool),
		failsOnUpd: make(map[string]bool),
	}
}

type ColNode struct {
	docNodes map[string]*DocNode
}

type DocNode struct {
	id       string
	data     interface{}
	colNodes map[string]*ColNode
}

type StoreMock struct {
	rootNode DocNode

	failsOnGet map[string]bool
	failsOnSet map[string]bool
	failsOnDel map[string]bool
	failsOnUpd map[string]bool
}

func (s *StoreMock) FailsOnGet(path string, fails bool) {
	s.failsOnGet[path] = false
}

func (s *StoreMock) FailsOnSet(path string, fails bool) {
	s.failsOnSet[path] = false
}

func (s *StoreMock) FailsOnDel(path string, fails bool) {
	s.failsOnDel[path] = false
}

func (s *StoreMock) FailsOnUpdate(path string, fails bool) {
	s.failsOnUpd[path] = false
}

func (s *StoreMock) Collection(name string) CollectionRef {
	colNode := s.rootNode.colNodes[name]

	if colNode == nil {
		colNode = &ColNode{
			docNodes: make(map[string]*DocNode),
		}

		s.rootNode.colNodes[name] = colNode
	}

	return &CollectionRefMock{
		storeMock: s,
		colPath:   []string{name},
		colNode:   colNode,
	}
}

func (s *StoreMock) Batch() Batch {
	return &BatchMock{
		elements: make([]*BatchElement, 0, 10),
	}
}

type CollectionRefMock struct {
	storeMock *StoreMock

	colPath []string
	colNode *ColNode
}

func (c *CollectionRefMock) Doc(id string) DocumentRef {
	docNode := c.colNode.docNodes[id]

	if docNode == nil {
		docNode = &DocNode{
			id:       id,
			data:     nil,
			colNodes: make(map[string]*ColNode),
		}

		c.colNode.docNodes[id] = docNode
	}

	return &DocumentRefMock{
		storeMock: c.storeMock,
		docPath:   append(c.colPath, id),
		docNode:   docNode,
	}
}

func (c *CollectionRefMock) Path() []string {
	return c.colPath
}

type DocumentRefMock struct {
	storeMock *StoreMock

	docPath []string
	docNode *DocNode
}

func (d *DocumentRefMock) Collection(name string) CollectionRef {
	colNode := d.docNode.colNodes[name]

	if colNode == nil {
		colNode = &ColNode{
			docNodes: make(map[string]*DocNode),
		}

		d.docNode.colNodes[name] = colNode
	}

	return &CollectionRefMock{
		storeMock: d.storeMock,
		colPath:   append(d.docPath, name),
		colNode:   colNode,
	}
}

func (d *DocumentRefMock) Path() []string {
	return d.docPath
}

func (d *DocumentRefMock) Set(dst interface{}) error {
	if d.storeMock.failsOnSet[strings.Join(d.Path(), "/")] {
		return errors.New("fails on set")
	}

	d.docNode.data = dst
	return nil
}

// TODO: MERGE FOR REAL
func (d *DocumentRefMock) Merge(dst interface{}) error {
	if d.storeMock.failsOnSet[strings.Join(d.Path(), "/")] {
		return errors.New("fails on set")
	}

	d.docNode.data = dst
	return nil
}

func (d *DocumentRefMock) Update(patch []firestore.Update) error {
	if d.storeMock.failsOnUpd[strings.Join(d.Path(), "/")] {
		return errors.New("fails on update")
	}

	if d.docNode.data == nil {
		return errors.New("doesn't exist")
	}

	for _, p := range patch {
		SetPathValue(d.docNode.data, p.Path, p.Value)
	}

	return nil
}

func (d *DocumentRefMock) Delete() error {
	if d.storeMock.failsOnDel[strings.Join(d.Path(), "/")] {
		return errors.New("fails on delete")
	}

	d.docNode.data = nil
	return nil
}

func (d *DocumentRefMock) Get(dst interface{}) (bool, error) {
	if d.storeMock.failsOnGet[strings.Join(d.Path(), "/")] {
		return false, errors.New("fails on get")
	}

	if d.docNode.data == nil {
		return false, nil
	}

	bytes, _ := json.Marshal(d.docNode.data)
	json.Unmarshal(bytes, dst)

	return true, nil
}

type BatchMock struct {
	elements []*BatchElement
}

type BatchElement struct {
	docRef DocumentRef
	method string
	data   interface{}
	patch  []firestore.Update
}

func (b *BatchMock) Set(ref DocumentRef, src interface{}) {
	b.elements = append(b.elements, &BatchElement{
		docRef: ref,
		method: "set",
		data:   src,
	})
}

func (b *BatchMock) Merge(ref DocumentRef, src interface{}) {
	b.elements = append(b.elements, &BatchElement{
		docRef: ref,
		method: "merge",
		data:   src,
	})
}

func (b *BatchMock) Update(ref DocumentRef, patch []firestore.Update) {
	b.elements = append(b.elements, &BatchElement{
		docRef: ref,
		method: "update",
		patch:  patch,
	})
}

func (b *BatchMock) Commit() error {

	for _, el := range b.elements {
		switch el.method {
		case "set":
			el.docRef.Set(el.data)
		case "update":
			el.docRef.Update(el.patch)
		case "merge":
			el.docRef.Merge(el.data)

		}
	}

	return nil
}

func GetPathValue(object interface{}, pathName string) interface{} {
	pathNames := strings.Split(pathName, ".")
	return getValue(object, pathNames)
}

func SetPathValue(object interface{}, pathName string, value interface{}) {
	path := strings.Split(pathName, ".")
	setValue(object, path, value)
}

func getValue(object interface{}, path []string) interface{} {
	if len(path) == 0 {
		return object
	}

	v := reflect.ValueOf(object)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Map {
		entry := v.MapIndex(reflect.ValueOf(path[0]))

		if !entry.IsValid() {
			return nil
		}

		return getValue(entry.Interface(), path[1:])
	}

	if v.Kind() == reflect.Struct {
		numField := v.NumField()

		for i := 0; i < numField; i++ {
			field := v.Type().Field(i)

			name := field.Tag.Get("firestore")

			if name == path[0] {
				return getValue(v.Field(i).Interface(), path[1:])
			}
		}
	}

	return nil
}

func setValue(object interface{}, path []string, value interface{}) {

	v := reflect.ValueOf(object)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Map {
		if len(path) > 1 {
			entry := v.MapIndex(reflect.ValueOf(path[0]))
			if !entry.IsValid() {
				return
			}

			setValue(entry.Addr().Interface(), path[1:], value)
			return
		}

		if reflect.TypeOf(value).String() == "firestore.sentinel" && reflect.ValueOf(value).IsZero() {
			v.SetMapIndex(reflect.ValueOf(path[0]), reflect.Value{})
		} else {
			v.SetMapIndex(reflect.ValueOf(path[0]), reflect.ValueOf(value))
		}

		return
	}

	if v.Kind() == reflect.Struct {
		numField := v.NumField()

		for i := 0; i < numField; i++ {
			field := v.Type().Field(i)

			name := field.Tag.Get("firestore")

			if name == path[0] {
				if len(path) > 1 {
					setValue(v.Field(i).Addr().Interface(), path[1:], value)
					return
				}

				v.Field(i).Set(reflect.ValueOf(value))
				return
			}
		}
	}

	panic("invalid path")
}
