package ddb

import "github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"

func And(left, right expression.ConditionBuilder, other ...expression.ConditionBuilder) expression.ConditionBuilder {
	return expression.And(left, right, other...)
}

func Not(cond expression.ConditionBuilder) expression.ConditionBuilder {
	return expression.Not(cond)
}

func Or(left, right expression.ConditionBuilder, other ...expression.ConditionBuilder) expression.ConditionBuilder {
	return expression.Or(left, right, other...)
}

func Eq(attribute string, value interface{}) expression.ConditionBuilder {
	return expression.Equal(expression.Name(attribute), expression.Value(value))
}

func NotEq(attribute string, value interface{}) expression.ConditionBuilder {
	return expression.NotEqual(expression.Name(attribute), expression.Value(value))
}

func Gt(attribute string, value interface{}) expression.ConditionBuilder {
	return expression.GreaterThan(expression.Name(attribute), expression.Value(value))
}

func Gte(attribute string, value interface{}) expression.ConditionBuilder {
	return expression.GreaterThanEqual(expression.Name(attribute), expression.Value(value))
}

func Lt(attribute string, value interface{}) expression.ConditionBuilder {
	return expression.LessThan(expression.Name(attribute), expression.Value(value))
}

func Lte(attribute string, value interface{}) expression.ConditionBuilder {
	return expression.LessThanEqual(expression.Name(attribute), expression.Value(value))
}

func Between(attribute string, lower interface{}, upper interface{}) expression.ConditionBuilder {
	return expression.Between(expression.Name(attribute), expression.Value(lower), expression.Value(upper))
}

func AttributeNotExists(attribute string) expression.ConditionBuilder {
	return expression.AttributeNotExists(expression.Name(attribute))
}

func AttributeExists(attribute string) expression.ConditionBuilder {
	return expression.AttributeExists(expression.Name(attribute))
}
