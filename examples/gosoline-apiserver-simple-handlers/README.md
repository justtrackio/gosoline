# Api Server examples
1. GET request returning JSON example
2. POST request with form input returning JSON example
3. Adding api key authentication
4. Adding crud operations

## 1. GET request returning JSON example
* Open two shells
* In the first shell type the following and wait for `kernel up and running`:
```bash
go run .
```
* In the second shell type:
```bash
curl http://127.0.0.1:8088/json-from-map
curl http://127.0.0.1:8088/json-from-struct
```
* both should reply with this:
```json
{"status":"success"}
```

## 2. POST request with form input returning JSON example
* Open two shells
* In the first shell type the following and wait for `kernel up and running`:
```bash
go run .
```
* In the second shell type:
```bash
curl -XPOST http://127.0.0.1:8088/json-handler
```
* your output should be an error, because you didn't supply a body: 
```json
{"err":"EOF"}
```
* Now try this instead:
```bash
curl -XPOST http://127.0.0.1:8088/json-handler -d '{}'
```
* your output should be an error, but a different indeed because you didn't supply the required input variables for this route: 
```json
{"err":"Key: 'inputEntity.Message' Error:Field validation for 'Message' failed on the 'required' tag"}
```
* Now let's do a successful call to the route:
```bash
curl -XPOST http://127.0.0.1:8088/json-handler -d '{"message":"hello, please handle this"}'
```
* your output should be a json object with a message property: 
```json
{"message":"Thank you for submitting your message 'hello, please handle this', we will handle it with care!"}
```

## 3. using api-key/http-basic authentication
* Open two shells
* In the first shell type the following and wait for `kernel up and running`:
```bash
go run .
```
* In the second shell type:
```bash
curl http://127.0.0.1:8088/admin/authenticated
```
* your output should be an error, because you didn't supply an api key: 
```json
{"api-key":"no api key provided","basic-auth":"no credentials provided"}
```
* Now try this instead:
```bash
curl -H 'X-API-KEY: someKey' http://127.0.0.1:8088/admin/authenticated
```
* your output should be an error, but a different indeed because you didn't supply (one of) the correct api key(s): 
```json
{"api-key":"api key does not match","basic-auth":"no credentials provided"}
```
* Now let's do a successful call with an api key to the route:
```bash
curl -H 'X-API-KEY: changeMe' http://127.0.0.1:8088/admin/authenticated
```
* your output should be a json object with a message property: 
```json
{"authenticated":true}
```
* And now let's do a successful call with a basic auth to the route:
```bash
curl -H 'Authorization: Basic YWRtaW46cGFzc3dvcmQ=' http://127.0.0.1:8088/admin/authenticated
```
* your output should be a json object with a message property: 
```json
{"authenticated":true}
```

## 4. Adding crud operations
* Open two shells
* In the first shell type the following and wait for `kernel up and running`:
```bash
go run .
```
* In the second shell type:
```bash
curl -XPOST http://127.0.0.1:8088/v0/myEntities -d '{}'
```
* your output should be: 
```json
{"total":2,"results":[{"id":1,"prop1":"text","prop2":"","createdAt":null,"updatedAt":null},{"id":2,"prop1":"text","prop2":"","createdAt":null,"updatedAt":null}]}
```
* this is a demo just for listing. You can also create/read/update/delete with (not implemented in this test app):
```bash
# create
curl -XPOST http://127.0.0.1:8088/v0/myEntity -d '{"prop1:"test"}'
# read
curl http://127.0.0.1:8088/v0/myEntity/1
# update
curl -XPUT http://127.0.0.1:8088/v0/myEntity/1 -d '{"prop1:"test"}'
# delete
curl -XDELETE http://127.0.0.1:8088/v0/myEntity/1

```
