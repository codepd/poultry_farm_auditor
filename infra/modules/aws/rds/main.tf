locals {
  db_identifier = replace(lower(var.db_name), "_", "-")
}

resource "aws_db_subnet_group" "this" {
  name       = "${var.db_name}-subnet-group"
  subnet_ids = var.private_subnet_ids

  tags = merge(var.tags, {
    Name = "${var.db_name}-subnet-group"
  })
}

resource "aws_security_group" "rds" {
  name        = "${var.db_name}-rds-sg"
  description = "Allow Postgres access from private VPC range"
  vpc_id      = var.vpc_id

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [var.allowed_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, {
    Name = "${var.db_name}-rds-sg"
  })
}

resource "aws_db_instance" "this" {
  identifier             = "${local.db_identifier}-dev"
  engine                 = "postgres"
  engine_version         = "15.15"
  instance_class         = var.instance_class
  allocated_storage      = var.allocated_storage
  db_name                = var.db_name
  username               = var.db_username
  password               = var.db_password
  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  publicly_accessible    = false
  skip_final_snapshot    = true
  deletion_protection    = false

  tags = merge(var.tags, {
    Name = "${local.db_identifier}-db"
  })
}
