package store

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewStore(projectId string) (Store, error) {
	conf := &firebase.Config{ProjectID: projectId}

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, conf)

	if err != nil {
		return nil, errors.New("can't initialize firebase")
	}

	client, err := app.Firestore(ctx)

	if err != nil {
		return nil, errors.New("can't initialize firestore")
	}

	return &StoreImpl{
		ctx:    ctx,
		client: client,
	}, nil
}

type Store interface {
	Collection(name string) CollectionRef
	Batch() Batch
}

type StoreImpl struct {
	ctx    context.Context
	client *firestore.Client
}

func (s *StoreImpl) Collection(name string) CollectionRef {
	return &CollectionRefImpl{
		ctx:    s.ctx,
		colRef: s.client.Collection((name)),
		path:   []string{name},
	}
}

func (s *StoreImpl) Batch() Batch {
	return &BatchImpl{
		ctx:        s.ctx,
		client:     s.client,
		writeBatch: s.client.Batch(),
	}
}

type CollectionRef interface {
	Doc(id string) DocumentRef

	Path() []string
}

type CollectionRefImpl struct {
	path   []string
	ctx    context.Context
	colRef *firestore.CollectionRef
}

func (c *CollectionRefImpl) Doc(id string) DocumentRef {
	return &DocumentRefImpl{
		path:   append(c.path, id),
		ctx:    c.ctx,
		docRef: c.colRef.Doc(id),
	}
}

func (c *CollectionRefImpl) Path() []string {
	return c.path
}

type DocumentRef interface {
	Collection(name string) CollectionRef
	Get(dst interface{}) (bool, error)
	Set(dst interface{}) error
	Update(patch []firestore.Update) error
	Delete() error
	Path() []string
}

type DocumentRefImpl struct {
	path   []string
	ctx    context.Context
	docRef *firestore.DocumentRef
}

func (d *DocumentRefImpl) Collection(name string) CollectionRef {
	return &CollectionRefImpl{
		ctx:    d.ctx,
		colRef: d.docRef.Collection(name),
		path:   append(d.path, name),
	}
}

func (d *DocumentRefImpl) Path() []string {
	return d.path
}

func (d *DocumentRefImpl) Set(src interface{}) error {
	_, err := d.docRef.Set(d.ctx, src)
	return err
}

func (d *DocumentRefImpl) Update(src []firestore.Update) error {
	_, err := d.docRef.Update(d.ctx, src)
	return err
}

func (d *DocumentRefImpl) Delete() error {
	_, err := d.docRef.Delete(d.ctx)
	return err
}

func (d *DocumentRefImpl) Get(dst interface{}) (bool, error) {
	snap, err := d.docRef.Get(d.ctx)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}

		return false, err
	}

	snap.DataTo(dst)
	return true, nil
}

type Batch interface {
	Set(ref DocumentRef, src interface{})
	Update(ref DocumentRef, patch []firestore.Update)
	Commit() error
}

type BatchImpl struct {
	ctx        context.Context
	client     *firestore.Client
	writeBatch *firestore.WriteBatch
}

func (b *BatchImpl) Set(ref DocumentRef, src interface{}) {
	documentRef := toFirestoreDocRef(*b.client, ref)
	b.writeBatch.Set(documentRef, src)
}

func (b *BatchImpl) Update(ref DocumentRef, patch []firestore.Update) {
	documentRef := toFirestoreDocRef(*b.client, ref)
	b.writeBatch.Update(documentRef, patch)
}

func (b *BatchImpl) Commit() error {
	_, err := b.writeBatch.Commit(b.ctx)
	return err
}

type Query interface {
	Where(id string) Query
	Limit(limit int)
}

func toFirestoreDocRef(client firestore.Client, docRef DocumentRef) *firestore.DocumentRef {

	path := docRef.Path()
	pathLength := len(path)

	if pathLength == 0 || pathLength%2 == 1 {
		return nil
	}

	documentRef := client.Collection(path[0]).Doc(path[1])

	for i := 2; i < pathLength; i += 2 {
		documentRef = documentRef.Collection(path[0]).Doc(path[1])
	}

	return documentRef
}
