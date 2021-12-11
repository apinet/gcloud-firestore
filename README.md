# gcloud-firestore

Firestore (gcloud) wrapper with mocking capabilities for golang cloud functions.

## usage

### initiate store

```go
import (
  store "github.com/apinet/gcloud-firestore"
)

s := store.NewStore("project-id")
```

### document operations

```go
leffe := Leffe{}
found, err := s.Collection("beer").Doc("leffe").Get(&leffe)
err := s.Collection("beer").Doc("leffe").Set(&leffe)
err := s.Collection("beer").Doc("leffe").Update(patch)
err := s.Collection("beer").Doc("leffe").Delete()
```

### batch operations

```go
batch := s.Batch()
batch.Set(doc1Ref, &leffe)
batch.Update(doc2Ref, &patch)
err := batch.Commit()
```
