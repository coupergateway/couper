server "spa" {
  spa "spa1" {
    base_path = "/"
    bootstrap_file = "01_app.html"
    paths = ["/**"]
  }
  spa "spa2" {
    base_path = "/spa"
    bootstrap_file = "01_app.html"
    paths = ["/**"]
  }
  api {
  }
}
