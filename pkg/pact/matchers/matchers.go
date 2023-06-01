package matchers

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Matcher interface {
	isMatcher()
}

type Obj map[string]Matcher

func (m Obj) isMatcher() {}

func EqualTo[T json.Marshaler](example T) Matcher {
	return equalTo{
		example: example,
	}
}

type equalTo struct {
	example json.Marshaler
}
func (e equalTo) isMatcher() {}
func (e equalTo) MarshalJSON() ([]byte, error) {
	ex, err := json.Marshal(e.example)
	if err != nil {
		return  nil, err
	}

	s := fmt.Sprintf(`"matching(equalTo, '%s')"`, string(ex))

	return []byte(s), nil
}
func Int32(example int) Matcher {
	return int32Matcher{
		example: example,
	}
}

type int32Matcher struct {
	example int
}

func (i int32Matcher) isMatcher() {}
func (i int32Matcher) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf(`"matching(integer, %d)"`, i.example)

	return []byte(s), nil
}


func Decimal(example float64) Matcher {
	return decimal{
		example: example,
	}
}

type decimal struct {
	example float64
}

func (d decimal) isMatcher() {}
func (d decimal) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf(`"matching(decimal, %f)"`, d.example)

	return []byte(s), nil
}


func Int64(example int64) Matcher {
	return int64Matcher{
		example: example,
	}
}

type int64Matcher struct {
	example int64
}

func (i int64Matcher) isMatcher() {}
func (i int64Matcher) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf(`"matching(number, %d)"`, i.example)

	return []byte(s), nil
}

func Regex(exp regexp.Regexp, example string) Matcher {
	return regex{
		exp: exp,
		example: example,
	}
}

type regex struct {
	exp     regexp.Regexp
	example string
}

func (r regex) isMatcher() {}
func (r regex) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf(`"matching(regex, '%s', '%s')"`, r.exp.String(), r.example)

	return []byte(s), nil
}

func Timestamp(example time.Time) Matcher {
	return timestamp{
		example: example,
	}
}

type timestamp struct {
	example time.Time
}

func (t timestamp) isMatcher() {}

func (t timestamp) MarshalJSON() ([]byte, error) {
	ts := Obj{
		"seconds": Int64(t.example.Unix()),
		"nanos":   Int32(t.example.Nanosecond()),
	}

	return json.Marshal(ts)
}

type stringMatcher struct {
	example string
}

func Str(example string) Matcher {
	return stringMatcher{
		example: example,
	}
}

func (s stringMatcher) isMatcher() {}
func (s stringMatcher) MarshalJSON() ([]byte, error) {
	st := fmt.Sprintf(`"%s"`, s.example)

	return []byte(st), nil
}

type array struct {
	example []Matcher
}

func Arr(example ...Matcher) Matcher {
	return array{
		example: example,
	}
}

func (a array) isMatcher() {}

//func (a array) MarshalJSON() ([]byte, error) {
//	examples := make([]string, len(a.example))
//
//	for i, e := range a.example {
//		ex, err := json.Marshal(e)
//		if err != nil {
//			return nil, err
//		}
//
//		examples[i] = string(ex)
//	}
//
//	s := fmt.Sprintf(`"matching(array(%s))"`, join(examples, ", "))
//
//	return []byte(s), nil
//}

func EachValue(matcher Matcher) Matcher {
	return eachValue{
		matcher: matcher,
	}
}

type eachValue struct {
	matcher Matcher
}

func (e eachValue) isMatcher() {}

func (e eachValue) MarshalJSON() ([]byte, error) {
	ex, err := json.Marshal(e.matcher)
	if err != nil {
		return nil, err
	}

	trimmed := strings.Trim(string(ex), `"`)

	s := fmt.Sprintf(`"eachValue(%s)"`, trimmed)

	return []byte(s), nil
}