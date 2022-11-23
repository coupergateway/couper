server {
  endpoint "/" {
    response {
      json_body = [env.KEY1, env.KEY2, env.KEY3, env.KEY4, env.KEY5]
    }
  }
}

defaults {
  environment_variables = {
    KEY1 = "value1"
    KEY2 = env.VALUE_2
    KEY3 = default(env.VALUE_3, "default_value_3")
    KEY4 = env.VALUE_4
    "KEY5" = "value5"
  }
}
