server {
  spa {
    bootstrap_file = "./02_spa.hcl"
    paths = ["/app"]
  }

  spa "another" {
    bootstrap_file = "./02_spa.hcl"
    paths = ["/another"]
  }
}
