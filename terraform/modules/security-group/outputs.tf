output "security_group_id" {
  description = "The ID of the worker management security group"
  value       = aws_security_group.github_event_all_worker_mgmt_sg.id
}
