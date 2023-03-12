vyos_host = "router.example.com"
return {
  sources = {
    vyos = {
      "vyos_ssh",
      config = {
        host = vyos_host,
      },
    },
  },
  providers = {
    route53 = {
      "aws_route53",
      config = {
        record_suffix = ".example.com",
        forward_zone_id = "Z2FDTNDATAQYW2",
      },
    },
  },
}
