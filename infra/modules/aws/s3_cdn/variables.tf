variable "bucket_name" {
  type = string
}

variable "domain_name" {
  type     = string
  default  = ""
  nullable = true
}

variable "route53_zone_id" {
  type     = string
  default  = ""
  nullable = true
}

variable "tags" {
  type = map(string)
}
