server {
  spa "s1" {
    bootstrap_file = "couper.hcl"
    base_path = "/a"
    paths = ["/b/c/**"]
  }
  spa "s2" {
    bootstrap_file = "couper.hcl"
    base_path = "/a/b"
    paths = ["/c/**"]
  }
}
