output "ecs_cluster_id" {
  value = aws_ecs_cluster.main.id
}

output "ecs_cluster_arn" {
  value = aws_ecs_cluster.main.arn
}

output "iam_task_role_arn" {
  value = aws_iam_role.ecs-container.arn
}

output "iam_task_role_name" {
  value = aws_iam_role.ecs-container.name
}

output "iam_events_role_arn" {
  value = aws_iam_role.ecs-events.arn
}

output "iam_events_role_name" {
  value = aws_iam_role.ecs-events.name
}
