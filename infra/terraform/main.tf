# =============================================================================
# PudimProductivity — Infrastructure (Terraform)
# =============================================================================
# NOTE: This file is a skeleton for future AWS EKS + RDS deployment.
# It is NOT active yet. Uncomment and configure when ready to deploy.
#
# Prerequisites:
#   - AWS CLI configured with appropriate credentials
#   - Terraform >= 1.6
#   - S3 bucket for remote state (update backend config below)
# =============================================================================

# ---------------------------------------------------------------------------
# Remote State (uncomment when ready)
# ---------------------------------------------------------------------------
# terraform {
#   backend "s3" {
#     bucket = "pudimproductivity-terraform-state"
#     key    = "infra/terraform.tfstate"
#     region = "us-east-1"
#   }
# }

# ---------------------------------------------------------------------------
# Providers
# ---------------------------------------------------------------------------
# provider "aws" {
#   region = var.aws_region
# }

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------
# variable "aws_region" {
#   description = "AWS region for all resources"
#   type        = string
#   default     = "us-east-1"
# }

# variable "environment" {
#   description = "Deployment environment (dev, staging, prod)"
#   type        = string
#   default     = "dev"
# }

# variable "app_name" {
#   description = "Application name used for resource naming"
#   type        = string
#   default     = "pudimproductivity"
# }

# ---------------------------------------------------------------------------
# Networking — VPC
# ---------------------------------------------------------------------------
# resource "aws_vpc" "main" {
#   cidr_block           = "10.0.0.0/16"
#   enable_dns_hostnames = true
#   enable_dns_support   = true
#
#   tags = {
#     Name        = "${var.app_name}-${var.environment}-vpc"
#     Environment = var.environment
#   }
# }

# resource "aws_subnet" "public" {
#   count             = 2
#   vpc_id            = aws_vpc.main.id
#   cidr_block        = cidrsubnet(aws_vpc.main.cidr_block, 8, count.index)
#   availability_zone = data.aws_availability_zones.available.names[count.index]
#   map_public_ip_on_launch = true
#
#   tags = {
#     Name        = "${var.app_name}-${var.environment}-public-${count.index}"
#     Environment = var.environment
#   }
# }

# resource "aws_subnet" "private" {
#   count             = 2
#   vpc_id            = aws_vpc.main.id
#   cidr_block        = cidrsubnet(aws_vpc.main.cidr_block, 8, count.index + 2)
#   availability_zone = data.aws_availability_zones.available.names[count.index]
#
#   tags = {
#     Name        = "${var.app_name}-${var.environment}-private-${count.index}"
#     Environment = var.environment
#   }
# }

# data "aws_availability_zones" "available" {
#   state = "available"
# }

# ---------------------------------------------------------------------------
# RDS — PostgreSQL
# ---------------------------------------------------------------------------
# resource "aws_db_subnet_group" "main" {
#   name       = "${var.app_name}-${var.environment}-db-subnet-group"
#   subnet_ids = aws_subnet.private[*].id
#
#   tags = {
#     Name        = "${var.app_name}-${var.environment}-db-subnet-group"
#     Environment = var.environment
#   }
# }

# resource "aws_db_instance" "postgres" {
#   identifier             = "${var.app_name}-${var.environment}-db"
#   engine                 = "postgres"
#   engine_version         = "16.3"
#   instance_class         = "db.t3.medium"
#   allocated_storage      = 20
#   storage_type           = "gp3"
#   db_name                = "pudimproductivity"
#   username               = "pudim"
#   password               = var.db_password
#   db_subnet_group_name   = aws_db_subnet_group.main.name
#   vpc_security_group_ids = [aws_security_group.rds.id]
#   skip_final_snapshot    = true
#   backup_retention_period = 7
#   backup_window          = "03:00-04:00"
#   maintenance_window     = "sun:04:00-sun:05:00"
#
#   tags = {
#     Name        = "${var.app_name}-${var.environment}-db"
#     Environment = var.environment
#   }
# }

# variable "db_password" {
#   description = "PostgreSQL master password"
#   type        = string
#   sensitive   = true
# }

# ---------------------------------------------------------------------------
# EKS — Kubernetes Cluster
# ---------------------------------------------------------------------------
# resource "aws_eks_cluster" "main" {
#   name     = "${var.app_name}-${var.environment}-cluster"
#   role_arn = aws_iam_role.eks_cluster.arn
#   version  = "1.30"
#
#   vpc_config {
#     subnet_ids = aws_subnet.public[*].id
#   }
#
#   tags = {
#     Name        = "${var.app_name}-${var.environment}-cluster"
#     Environment = var.environment
#   }
# }

# resource "aws_iam_role" "eks_cluster" {
#   name = "${var.app_name}-${var.environment}-eks-cluster-role"
#
#   assume_role_policy = jsonencode({
#     Version = "2012-10-17"
#     Statement = [{
#       Effect = "Allow"
#       Principal = {
#         Service = "eks.amazonaws.com"
#       }
#       Action = "sts:AssumeRole"
#     }]
#   })
# }

# resource "aws_iam_role_policy_attachment" "eks_cluster_policy" {
#   policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
#   role       = aws_iam_role.eks_cluster.name
# }

# resource "aws_iam_role_policy_attachment" "eks_service_policy" {
#   policy_arn = "arn:aws:iam::aws:policy/AmazonEKSServicePolicy"
#   role       = aws_iam_role.eks_cluster.name
# }

# resource "aws_eks_node_group" "main" {
#   cluster_name    = aws_eks_cluster.main.name
#   node_group_name = "${var.app_name}-${var.environment}-nodes"
#   node_role_arn   = aws_iam_role.eks_nodes.arn
#   subnet_ids      = aws_subnet.private[*].id
#
#   scaling_config {
#     desired_size = 2
#     min_size     = 1
#     max_size     = 5
#   }
#
#   instance_types = ["t3.medium"]
#
#   tags = {
#     Name        = "${var.app_name}-${var.environment}-nodes"
#     Environment = var.environment
#   }
# }

# resource "aws_iam_role" "eks_nodes" {
#   name = "${var.app_name}-${var.environment}-eks-node-role"
#
#   assume_role_policy = jsonencode({
#     Version = "2012-10-17"
#     Statement = [{
#       Effect = "Allow"
#       Principal = {
#         Service = "ec2.amazonaws.com"
#       }
#       Action = "sts:AssumeRole"
#     }]
#   })
# }

# resource "aws_iam_role_policy_attachment" "eks_worker_node" {
#   policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
#   role       = aws_iam_role.eks_nodes.name
# }

# resource "aws_iam_role_policy_attachment" "eks_cni_policy" {
#   policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
#   role       = aws_iam_role.eks_nodes.name
# }

# resource "aws_iam_role_policy_attachment" "ecr_read_only" {
#   policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
#   role       = aws_iam_role.eks_nodes.name
# }

# ---------------------------------------------------------------------------
# Security Groups
# ---------------------------------------------------------------------------
# resource "aws_security_group" "rds" {
#   name        = "${var.app_name}-${var.environment}-rds-sg"
#   description = "Allow PostgreSQL access from EKS nodes"
#   vpc_id      = aws_vpc.main.id
#
#   ingress {
#     from_port       = 5432
#     to_port         = 5432
#     protocol        = "tcp"
#     security_groups = [aws_eks_cluster.main.vpc_config[0].cluster_security_group_id]
#   }
#
#   tags = {
#     Name        = "${var.app_name}-${var.environment}-rds-sg"
#     Environment = var.environment
#   }
# }

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------
# output "eks_cluster_endpoint" {
#   value = aws_eks_cluster.main.endpoint
# }

# output "rds_endpoint" {
#   value = aws_db_instance.postgres.endpoint
# }
