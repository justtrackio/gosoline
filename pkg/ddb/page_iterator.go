package ddb

import "github.com/aws/aws-sdk-go/aws"

type pageIterator struct {
	limit *int64
	size  *int64
	count *int64
}

func (p *pageIterator) isDone() bool {
	if p.limit == nil {
		return false
	}

	return *p.limit == *p.count
}

func (p *pageIterator) advance(count *int64) *int64 {
	p.count = aws.Int64(*p.count + *count)

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

	return aws.Int64(nextPageSize)
}

func buildPageIterator(limit *int64, size *int64) *pageIterator {
	if limit != nil && size == nil {
		size = aws.Int64(*limit)
	}

	if limit != nil && *size > *limit {
		size = aws.Int64(*limit)
	}

	progress := &pageIterator{
		limit: limit,
		size:  size,
		count: aws.Int64(0),
	}

	return progress
}
