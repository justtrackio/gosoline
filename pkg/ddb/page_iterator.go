package ddb

import "github.com/aws/aws-sdk-go-v2/aws"

type pageIterator struct {
	limit *int32
	size  *int32
	count *int32
}

func (p *pageIterator) isDone() bool {
	if p.limit == nil {
		return false
	}

	return *p.limit == *p.count
}

func (p *pageIterator) advance(count *int32) *int32 {
	p.count = aws.Int32(*p.count + *count)

	if p.limit == nil && p.size == nil {
		return nil
	}

	if p.limit == nil && p.size != nil {
		return p.size
	}

	remainingCount := *p.limit - *p.count
	nextPageSize := *p.size

	if remainingCount < nextPageSize {
		nextPageSize = remainingCount
	}

	return aws.Int32(nextPageSize)
}

func buildPageIterator(limit *int32, size *int32) *pageIterator {
	if limit != nil && size == nil {
		size = aws.Int32(*limit)
	}

	if limit != nil && *size > *limit {
		size = limit
	}

	progress := &pageIterator{
		limit: limit,
		size:  size,
		count: aws.Int32(0),
	}

	return progress
}
