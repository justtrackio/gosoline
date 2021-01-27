output "queue_id" {
  value = module.main.id
}

output "queue_arn" {
  value = module.main.arn
}

output "queue_name" {
  value = module.main.name
}

output "dead_queue_id" {
  value = module.dead.id
}

output "dead_queue_arn" {
  value = module.dead.arn
}

output "dead_queue_name" {
  value = module.dead.name
}
