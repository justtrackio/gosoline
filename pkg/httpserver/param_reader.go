package httpserver

import (
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
