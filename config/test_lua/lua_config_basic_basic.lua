return {
  sources = {
    vyos = {
      "vyos_ssh",
      config = {
        host = os.getenv("VYOS_HOST"),
        username = os.getenv("VYOS_USERNAME"),
        password = os.getenv("VYOS_PASSWORD"),
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
