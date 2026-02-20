server "couper" {
  endpoint "/fixed" {
    proxy {
      backend = "RLFixed"
    }
  }
  endpoint "/sliding" {
    proxy {
      backend = "RLSliding"
    }
  }
  endpoint "/block" {
    proxy {
      backend = "RLBlock"
    }
  }
}

definitions {
  backend "RLFixed" {
    path = "/small"
    origin = env.COUPER_TEST_BACKEND_ADDR

    throttle {
      period        = "3s"
      per_period    = 2
      period_window = "fixed"
    }
  }
  backend "RLSliding" {
    path = "/small"
    origin = env.COUPER_TEST_BACKEND_ADDR

    throttle {
      period        = "3s"
      per_period    = 2
      period_window = "sliding"
    }
  }
  backend "RLBlock" {
    path = "/small"
    origin = env.COUPER_TEST_BACKEND_ADDR

    throttle {
      period        = "1s"
      per_period    = 2
      period_window = "fixed"
      mode          = "block"
    }
  }
}
