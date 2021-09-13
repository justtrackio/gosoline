package apiserver

import (
	"strconv"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

func GetInt64FromRequest(request *Request, name string) (*int64, bool) {
	paramString, found := request.Params.Get(name)

	if !found {
		return nil, false
	}

	param, err := strconv.ParseInt(paramString, 10, 64)
	if err != nil {
		return nil, false
	}

	return mdl.Int64(param), true
}

func GetStringFromRequest(request *Request, name string) (*string, bool) {
	paramString, found := request.Params.Get(name)

	if !found {
		return mdl.String(""), false
	}

	return &paramString, true
}
