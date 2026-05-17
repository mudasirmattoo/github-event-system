provider "aws" {
  region = "us-east-1"
}

module "vpc" {
  source = "./modules/vpc"
}

module "security_group" {
  source = "./modules/security-group"

  vpc_id = module.vpc.vpc_id
}

module "eks" {
  source = "./modules/eks"

  vpc_id                   = module.vpc.vpc_id
  subnet_ids               = module.vpc.private_subnets
  worker_security_group_id = module.security_group.security_group_id
}

resource "aws_ecr_repository" "github-events-repo" {
  name                 = "github-events-repo"
  image_tag_mutability = "MUTABLE"
  force_delete         = true
}

resource "aws_ecr_repository_policy" "github-events-repo-policy" {
  repository = aws_ecr_repository.github-events-repo.name
  policy = jsonencode({
    Version = "2008-10-17",
    Statement = [
      {
        Sid    = "AllowPushPull",
        Effect = "Allow",
        Principal = {
          AWS = "*"
        },
        Action = ["ecr:BatchGetImage", "ecr:BatchCheckLayerAvailability", "ecr:GetDownloadUrlForLayer", "ecr:GetAuthorizationToken"]
      }
    ]
  })
}
