server "spa" {
  spa "spa1" {
    base_path = "/"
    bootstrap_file = "01_app.html"
    paths = ["/**"]
  }
  api {
  }
}
