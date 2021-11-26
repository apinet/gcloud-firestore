# gcloud-firestore
Firestore (gcloud) wrapper with mocking capabilities

## usage

### initiate store
```go
store := firestore.NewStore("project-id")
```

### get/set document
```go
leffe := Leffe{}
found, err := store.Collection("beer").Doc("leffe").Get(&leffe)
err := store.Collection("beer").Doc("leffe").Set(&leffe)
```
