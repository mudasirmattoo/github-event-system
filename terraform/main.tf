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
  private_subnets          = module.vpc.private_subnets
  worker_security_group_id = module.security_group.security_group_id
}
