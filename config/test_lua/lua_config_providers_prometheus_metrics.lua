return {
  providers = {
    prom = {
      "prometheus_metrics",
      config = {
        source_labels = {
          "source",
        },
      },
    },
  },
}
