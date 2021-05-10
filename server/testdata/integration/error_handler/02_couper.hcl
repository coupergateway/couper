server "access_control" {
  endpoint "/" {
    access_control = ["ba"]
    response {
      status = 204
    }
  }
}

definitions {
  basic_auth "ba" {
    password = "couper"

    error_handler {
      response {} #default 200 OK
    }
  }
}
