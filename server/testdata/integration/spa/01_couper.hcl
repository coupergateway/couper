server "spa" {
  error_file = "./../server_error.html"
  spa {
    bootstrap_file = "app.html"
    paths = ["/"]
  }
}
