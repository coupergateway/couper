server "env" {
  access_control = [ "ba1", "ba2" ]
  disable_access_control = [
    env.BAU1 == "" ? "ba2": "",
    env.BAU2 == "" ? "ba1": ""
  ]

  endpoint "/" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
  }
}

definitions {
  basic_auth "ba1" {
    user     = env.BAU1
    password = env.BAP1
  }
  basic_auth "ba2" {
    user     = env.BAU2
    password = env.BAP2
  }
}

defaults {
  environment_variables = {
    BAU1 = "user1"
  }
}
