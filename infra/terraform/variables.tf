variable "instance_type" {
  description = "EC2 instance type for the docker-compose host"
  type        = string
  default     = "t3.small"
}

variable "db_instance_class" {
  description = "RDS Postgres instance class"
  type        = string
  default     = "db.t3.micro"
}

variable "key_name" {
  description = "Existing AWS key pair name for SSH into the app host"
  type        = string
}

variable "media_bucket_name" {
  description = "S3 bucket for media uploads (must be globally unique)"
  type        = string
  default     = "pudimproductivity-media"
}

variable "db_password" {
  description = "Postgres master password"
  type        = string
  sensitive   = true
}
