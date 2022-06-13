server "couper" {
  endpoint "/down" {
    proxy {
      url = env.NO_ORIGIN
    }
  }
}
