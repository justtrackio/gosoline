package httpserver

import (
	"fmt"
	"strconv"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

func GetUintFromRequest(request *Request, name string) (*uint, bool) {
	paramString, found := request.Params.Get(name)

	if !found {
		return mdl.Box(uint(0)), false
	}

	param, err := strconv.Atoi(paramString)
	if err != nil {
		return mdl.Box(uint(0)), false
	}

	return mdl.Box(uint(param)), true
}

func GetStringFromRequest(request *Request, name string) (*string, bool) {
	paramString, found := request.Params.Get(name)

	if !found {
		return mdl.Box(""), false
	}

	return &paramString, true
}

func GetIdentifierFromRequest[K mdl.PossibleIdentifier](request *Request, name string) (*K, bool) {
	var k K
	var ki any = k

	switch ki.(type) {
	case string:
		id, valid := GetStringFromRequest(request, name)
		var idVal any = id

		return idVal.(*K), valid
	case uint:
		id, valid := GetUintFromRequest(request, name)
		var idVal any = id

		return idVal.(*K), valid
	default:
		panic(fmt.Errorf("type K should either be uint or string, got %T", k))
	}
}
