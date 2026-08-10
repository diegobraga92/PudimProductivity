output "app_public_ip" {
  description = "Public IP of the docker-compose host"
  value       = aws_instance.app.public_ip
}

output "app_public_dns" {
  description = "Public DNS of the docker-compose host"
  value       = aws_instance.app.public_dns
}

output "postgres_endpoint" {
  description = "RDS Postgres endpoint (host:port)"
  value       = "${aws_db_instance.postgres.address}:${aws_db_instance.postgres.port}"
}

output "media_bucket" {
  description = "S3 media bucket name"
  value       = aws_s3_bucket.media.id
}
