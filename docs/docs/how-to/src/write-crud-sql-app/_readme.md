### create
```shell
curl -d '{"text":"do it!", "dueDate":"2023-09-08T15:00:00Z"}' -H "Content-Type: application/json" -X POST localhost:8080/v0/todo
```

### get
```shell
curl -X GET localhost:8080/v0/todo/1
```

### update
```shell
curl -d '{"text":"do it!!!"}' -H "Content-Type: application/json" -X PUT localhost:8080/v0/todo/1
```

## list
```shell
curl -d '{}' -X POST localhost:8080/v0/todos
```

### delete
```shell
curl -X DELETE localhost:8080/v0/todo/1
```