package ddb

import "github.com/aws/aws-sdk-go/aws"

type readProgress struct {
	limit    *int64
	pageSize *int64
	count    *int64
}

func (p *readProgress) isDone() bool {
	if p.limit == nil {
		return false
	}

	return *p.limit == *p.count
}

func (p *readProgress) advance(count *int64) *int64 {
	p.count = aws.Int64(*p.count + *count)

	if p.limit == nil && p.pageSize == nil {
		return nil
	}

	if p.limit == nil && p.pageSize != nil {
		return p.pageSize
	}

	remainingCount := *p.limit - *p.count
	nextPageSize := *p.pageSize

	if remainingCount < nextPageSize {
		nextPageSize = remainingCount
	}

	return aws.Int64(nextPageSize)
}

func buildProgress(limit *int64, pageSize *int64) *readProgress {
	if limit != nil && pageSize == nil {
		pageSize = aws.Int64(*limit)
	}

	if limit != nil && *pageSize > *limit {
		pageSize = aws.Int64(*limit)
	}

	progress := &readProgress{
		limit:    limit,
		pageSize: pageSize,
		count:    aws.Int64(0),
	}

	return progress
}
