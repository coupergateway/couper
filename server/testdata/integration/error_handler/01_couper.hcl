server "access_control" {

  endpoint "/" {
    access_control = ["ba"]
    response {
      status = 204
    }
  }

  endpoint "/default" {
    access_control = ["default"]
    response {
      status = 204
    }
  }
}

definitions {
  basic_auth "ba" {
    password = "couper"

    error_handler "basic_auth" {
      response {
        status = 404
      }
    }

    error_handler "basic_auth_credentials_required" {
      response {
        status = 502
      }
    }
  }

  basic_auth "default" {
    password = "couper"
    realm = "protected"
  }
}
