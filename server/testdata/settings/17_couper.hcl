server {
  endpoint "/" {
	proxy {
		backend {
			beta_rate_limit {
				period = "1m"
				per_period = 60
			}
		}
	}
  }
}