server {
  access_control = ["ba"]

  api {
    endpoint "/**" {
      response {
        status = 204
      }
    }
  }
}

definitions {
  basic_auth "ba" {
    user = "u"
    password = "p"
  }
}
