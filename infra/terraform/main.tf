# PudimProductivity — Infrastructure (Terraform)
# =============================================================================
# Minimal single-host deployment: EC2 runs docker-compose (per ADR 006), S3
# holds media uploads, RDS hosts Postgres. `terraform validate` passes without
# credentials; `terraform apply` requires the AWS CLI configured.
#
#   cd infra/terraform
#   terraform init
#   terraform validate
#   terraform plan    # review
#   terraform apply   # provision
# =============================================================================

terraform {
  required_version = ">= 1.6"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  # Uncomment for remote state (create the S3 bucket first):
  # backend "s3" {
  #   bucket = "pudimproductivity-terraform-state"
  #   key    = "infra/terraform.tfstate"
  #   region = "us-east-1"
  # }
}

# Use the default VPC + subnets to keep the skeleton small. A full network
# design (dedicated VPC, NAT, private subnets) is a documented follow-up.
data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

# --- Security groups ---------------------------------------------------------

resource "aws_security_group" "app" {
  name        = "pudim-app"
  description = "HTTP/HTTPS + SSH for the PudimProductivity host"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    description = "Backend API (docker-compose)"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "db" {
  name        = "pudim-db"
  description = "Postgres access from the app host only"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description     = "Postgres from app host"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.app.id]
  }
}

# --- Compute (single host, docker-compose per ADR 006) -----------------------

resource "aws_instance" "app" {
  ami                         = data.aws_ami.amazon_linux_2023.id
  instance_type               = var.instance_type
  vpc_security_group_ids      = [aws_security_group.app.id]
  associate_public_ip_address = true
  key_name                    = var.key_name
  user_data                   = <<-EOT
    #!/bin/bash
    set -euxo pipefail
    dnf install -y docker git
    systemctl enable --now docker
    curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
    git clone https://github.com/diegobraga92/PudimProductivity.git /opt/pudim || true
    cd /opt/pudim
    DATABASE_URL="postgres://pudim:${var.db_password}@${aws_db_instance.postgres.address}:5432/pudimproductivity?sslmode=disable" \
      docker-compose up -d
  EOT

  tags = { Name = "pudim-app" }
}

data "aws_ami" "amazon_linux_2023" {
  most_recent = true
  owners      = ["amazon"]
  filter {
    name   = "name"
    values = ["al2023-ami-2023.*-x86_64"]
  }
}

# --- Storage (media uploads) -------------------------------------------------

resource "aws_s3_bucket" "media" {
  bucket        = var.media_bucket_name
  force_destroy = true
}

resource "aws_s3_bucket_versioning" "media" {
  bucket = aws_s3_bucket.media.id
  versioning_configuration {
    status = "Enabled"
  }
}

# --- Database (Postgres) -----------------------------------------------------

resource "aws_db_instance" "postgres" {
  identifier             = "pudim-postgres"
  engine                 = "postgres"
  engine_version         = "16.4"
  instance_class         = var.db_instance_class
  allocated_storage      = 20
  db_name                = "pudimproductivity"
  username               = "pudim"
  password               = var.db_password
  vpc_security_group_ids = [aws_security_group.db.id]
  db_subnet_group_name   = aws_db_subnet_group.main.name
  skip_final_snapshot    = true
  multi_az               = false
}

resource "aws_db_subnet_group" "main" {
  name       = "pudim-main"
  subnet_ids = data.aws_subnets.default.ids
}

