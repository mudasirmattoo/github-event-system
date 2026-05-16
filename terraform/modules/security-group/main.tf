
resource "aws_security_group" "github_event_all_worker_mgmt_sg" {
  name        = "github-event-system-all-worker-mgmt-sg"
  description = "Security group for github event system all worker management"
  vpc_id      = var.vpc_id

}


resource "aws_security_group_rule" "all_worker_mgmt_ingress" {
  description = "allow inbound traffic from eks"

  type              = "ingress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  security_group_id = aws_security_group.github_event_all_worker_mgmt_sg.id
  cidr_blocks = [
    "10.0.0.0/8",
    "172.16.0.0/12",
    "192.168.0.0/16",
  ]
}


resource "aws_security_group_rule" "all_worker_mgmt_egress" {
  description       = "allow outbound traffic to anywhere"
  from_port         = 0
  protocol          = "-1"
  security_group_id = aws_security_group.github_event_all_worker_mgmt_sg.id
  to_port           = 0
  type              = "egress"
  cidr_blocks       = ["0.0.0.0/0"]
}