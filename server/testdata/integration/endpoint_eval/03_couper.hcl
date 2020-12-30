server "api" {
  api {

    backend {
      origin = env.COUPER_TEST_BACKEND_ADDR != "" ? env.COUPER_TEST_BACKEND_ADDR : "http://127.0.0.1:1"
      path = "/unset/by/endpoint"
    }

//    endpoint "/endpoint1" {
//      path = "/anything"
//    }
//
//    endpoint "/endpoint2" {
//     path = "/unset/by/local-backend"
//     backend "anything" {
//       path = "/anything"
//     }
//    }
//
//    # don't override path
//    endpoint "/endpoint3" {
//      backend  = "anything"
//    }

    endpoint "/endpoint4" {
      path = "/anything"
    }

  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/unset/by/endpoint"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
}
