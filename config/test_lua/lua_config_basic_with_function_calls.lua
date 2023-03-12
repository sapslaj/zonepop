function get_vyos_host ()
  return os.getenv("VYOS_HOST")
end
return {
  sources = {
    vyos = {
      "vyos_ssh",
      config = {
        host = get_vyos_host(),
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
